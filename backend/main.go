package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	_ "github.com/lib/pq"
)

// Gasto representa un gasto personal cargado por el usuario.
type Gasto struct {
	ID          int       `json:"id"`
	Descripcion string    `json:"descripcion"`
	Monto       float64   `json:"monto"`
	Categoria   string    `json:"categoria"`
	Fecha       time.Time `json:"fecha"`
}

// ResumenCategoria es el total gastado en una categoria.
type ResumenCategoria struct {
	Categoria string  `json:"categoria"`
	Total     float64 `json:"total"`
}

var db *sql.DB

func main() {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		log.Fatal("falta la variable de entorno DATABASE_URL")
	}

	var err error
	db, err = sql.Open("postgres", dsn)
	if err != nil {
		log.Fatalf("no se pudo abrir la conexion: %v", err)
	}
	defer db.Close()

	// El contenedor de la base puede tardar unos segundos en aceptar
	// conexiones incluso despues de que compose lo marque "running".
	if err := waitForDB(db, 15); err != nil {
		log.Fatalf("la base de datos no respondio a tiempo: %v", err)
	}

	if err := migrate(db); err != nil {
		log.Fatalf("no se pudo crear el schema: %v", err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", healthHandler)
	mux.HandleFunc("GET /api/gastos", listGastosHandler)
	mux.HandleFunc("POST /api/gastos", createGastoHandler)
	mux.HandleFunc("DELETE /api/gastos/{id}", deleteGastoHandler)
	mux.HandleFunc("GET /api/resumen", resumenHandler)

	log.Println("escuchando en :8080")
	if err := http.ListenAndServe(":8080", withCORS(mux)); err != nil {
		log.Fatal(err)
	}
}

// withCORS habilita llamadas desde el servidor de desarrollo de Vite
// (localhost:5173) cuando se corre el backend suelto, fuera de compose.
// Dentro de compose no hace falta (mismo origen via nginx), pero no molesta.
func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func waitForDB(db *sql.DB, attempts int) error {
	var err error
	for i := 0; i < attempts; i++ {
		if err = db.Ping(); err == nil {
			return nil
		}
		time.Sleep(2 * time.Second)
	}
	return err
}

func migrate(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS gastos (
			id SERIAL PRIMARY KEY,
			descripcion TEXT NOT NULL,
			monto NUMERIC(12,2) NOT NULL,
			categoria TEXT NOT NULL,
			fecha DATE NOT NULL DEFAULT CURRENT_DATE
		);
	`)
	return err
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	if err := db.Ping(); err != nil {
		http.Error(w, `{"status":"error"}`, http.StatusServiceUnavailable)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"status":"ok"}`))
}

func listGastosHandler(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query(`SELECT id, descripcion, monto, categoria, fecha FROM gastos ORDER BY fecha DESC, id DESC`)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	gastos := []Gasto{}
	for rows.Next() {
		var g Gasto
		if err := rows.Scan(&g.ID, &g.Descripcion, &g.Monto, &g.Categoria, &g.Fecha); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		gastos = append(gastos, g)
	}
	writeJSON(w, gastos)
}

func createGastoHandler(w http.ResponseWriter, r *http.Request) {
	var g Gasto
	if err := json.NewDecoder(r.Body).Decode(&g); err != nil {
		http.Error(w, "body invalido", http.StatusBadRequest)
		return
	}
	if g.Descripcion == "" || g.Categoria == "" || g.Monto <= 0 {
		http.Error(w, "descripcion, categoria y monto (mayor a 0) son obligatorios", http.StatusBadRequest)
		return
	}
	if g.Fecha.IsZero() {
		g.Fecha = time.Now()
	}

	err := db.QueryRow(
		`INSERT INTO gastos (descripcion, monto, categoria, fecha) VALUES ($1, $2, $3, $4) RETURNING id`,
		g.Descripcion, g.Monto, g.Categoria, g.Fecha,
	).Scan(&g.ID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusCreated)
	writeJSON(w, g)
}

func deleteGastoHandler(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "id invalido", http.StatusBadRequest)
		return
	}
	res, err := db.Exec(`DELETE FROM gastos WHERE id = $1`, id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		http.Error(w, "no existe ese gasto", http.StatusNotFound)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func resumenHandler(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query(`SELECT categoria, SUM(monto) FROM gastos GROUP BY categoria ORDER BY SUM(monto) DESC`)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	resumen := []ResumenCategoria{}
	for rows.Next() {
		var rc ResumenCategoria
		if err := rows.Scan(&rc.Categoria, &rc.Total); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		resumen = append(resumen, rc)
	}
	writeJSON(w, resumen)
}

func writeJSON(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}
