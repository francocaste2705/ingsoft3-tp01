package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"
	"github.com/paquete-que-no-existe/nada"
	_ "github.com/lib/pq"
)

// Gasto representa un gasto personal cargado por el usuario.
type Gasto struct {
	ID          int       `json:"id"`
	Descripcion string    `json:"descripcion"`
	Monto       float64   `json:"monto"`
	Categoria   string    `json:"categoria"`
	Fecha       time.Time `json:"fecha"`
	Estado      string    `json:"estado"`
}

// Presupuesto es el tope mensual definido para una categoria.
type Presupuesto struct {
	Categoria    string  `json:"categoria"`
	MontoMensual float64 `json:"monto_mensual"`
}

// ResumenCategoria es el total gastado en una categoria, con su presupuesto si tiene.
type ResumenCategoria struct {
	Categoria          string   `json:"categoria"`
	Total              float64  `json:"total"`
	PresupuestoMensual *float64 `json:"presupuesto_mensual,omitempty"`
	PorcentajeUsado    *float64 `json:"porcentaje_usado,omitempty"`
}

// CategoriasValidas es la lista cerrada de categorias aceptadas. Se valida
// en el servidor, no solo en el combo del frontend.
var CategoriasValidas = map[string]bool{
	"Comida":     true,
	"Transporte": true,
	"Ocio":       true,
	"Servicios":  true,
	"Salud":      true,
	"Otros":      true,
}

const (
	EstadoPendiente = "pendiente"
	EstadoPagado    = "pagado"
)

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
	mux.HandleFunc("PATCH /api/gastos/{id}/estado", updateEstadoHandler)
	mux.HandleFunc("GET /api/resumen", resumenHandler)
	mux.HandleFunc("GET /api/presupuestos", listPresupuestosHandler)
	mux.HandleFunc("POST /api/presupuestos", upsertPresupuestoHandler)

	log.Println("escuchando en :8080")
	if err := http.ListenAndServe(":8080", withCORS(mux)); err != nil {
		log.Fatal(err)
	}
}

func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, DELETE, OPTIONS")
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
			fecha DATE NOT NULL DEFAULT CURRENT_DATE,
			estado TEXT NOT NULL DEFAULT 'pendiente'
		);

		CREATE TABLE IF NOT EXISTS presupuestos (
			categoria TEXT PRIMARY KEY,
			monto_mensual NUMERIC(12,2) NOT NULL
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
	rows, err := db.Query(`SELECT id, descripcion, monto, categoria, fecha, estado FROM gastos ORDER BY fecha DESC, id DESC`)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	gastos := []Gasto{}
	for rows.Next() {
		var g Gasto
		if err := rows.Scan(&g.ID, &g.Descripcion, &g.Monto, &g.Categoria, &g.Fecha, &g.Estado); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		gastos = append(gastos, g)
	}
	if err := rows.Err(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, gastos)
}

func createGastoHandler(w http.ResponseWriter, r *http.Request) {
	var g Gasto
	if err := json.NewDecoder(r.Body).Decode(&g); err != nil {
		http.Error(w, "body invalido", http.StatusBadRequest)
		return
	}

	// Regla 1: campos obligatorios y monto positivo.
	if g.Descripcion == "" || g.Categoria == "" || g.Monto <= 0 {
		http.Error(w, "descripcion, categoria y monto (mayor a 0) son obligatorios", http.StatusBadRequest)
		return
	}

	// Regla 2: la categoria tiene que ser una de las validas (no lo que
	// mande el cliente, aunque el combo del front ya la restrinja).
	if !CategoriasValidas[g.Categoria] {
		http.Error(w, "categoria invalida", http.StatusBadRequest)
		return
	}

	if g.Fecha.IsZero() {
		g.Fecha = time.Now()
	}

	// Regla 3: no se aceptan gastos con fecha futura.
	hoy := truncarADia(time.Now())
	if truncarADia(g.Fecha).After(hoy) {
		http.Error(w, "la fecha del gasto no puede ser futura", http.StatusBadRequest)
		return
	}

	g.Estado = EstadoPendiente

	// Regla 4: si la categoria tiene presupuesto mensual definido, un
	// gasto que lo supere se bloquea. Sin presupuesto definido, no hay
	// restriccion (se permite cualquier monto).
	presupuesto, tienePresupuesto, err := presupuestoDeCategoria(g.Categoria)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if tienePresupuesto {
		totalDelMes, err := totalGastadoEnElMes(g.Categoria, g.Fecha)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if totalDelMes+g.Monto > presupuesto {
			http.Error(w, "el gasto excede el presupuesto mensual de la categoria", http.StatusUnprocessableEntity)
			return
		}
	}

	err = db.QueryRow(
		`INSERT INTO gastos (descripcion, monto, categoria, fecha, estado) VALUES ($1, $2, $3, $4, $5) RETURNING id`,
		g.Descripcion, g.Monto, g.Categoria, g.Fecha, g.Estado,
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

// Regla 5: transicion de estado. Un gasto pendiente puede pasar a pagado,
// pero un gasto pagado no puede volver a pendiente.
func updateEstadoHandler(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "id invalido", http.StatusBadRequest)
		return
	}

	var body struct {
		Estado string `json:"estado"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "body invalido", http.StatusBadRequest)
		return
	}
	if body.Estado != EstadoPendiente && body.Estado != EstadoPagado {
		http.Error(w, "estado invalido: debe ser 'pendiente' o 'pagado'", http.StatusBadRequest)
		return
	}

	var estadoActual string
	if err := db.QueryRow(`SELECT estado FROM gastos WHERE id = $1`, id).Scan(&estadoActual); err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "no existe ese gasto", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if estadoActual == EstadoPagado && body.Estado == EstadoPendiente {
		http.Error(w, "un gasto pagado no puede volver a pendiente", http.StatusConflict)
		return
	}

	if _, err := db.Exec(`UPDATE gastos SET estado = $1 WHERE id = $2`, body.Estado, id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, map[string]string{"id": idStr, "estado": body.Estado})
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

		presupuesto, tiene, err := presupuestoDeCategoria(rc.Categoria)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if tiene {
			rc.PresupuestoMensual = &presupuesto
			porcentaje := (rc.Total / presupuesto) * 100
			rc.PorcentajeUsado = &porcentaje
		}

		resumen = append(resumen, rc)
	}
	if err := rows.Err(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, resumen)
}

func listPresupuestosHandler(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query(`SELECT categoria, monto_mensual FROM presupuestos ORDER BY categoria`)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	presupuestos := []Presupuesto{}
	for rows.Next() {
		var p Presupuesto
		if err := rows.Scan(&p.Categoria, &p.MontoMensual); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		presupuestos = append(presupuestos, p)
	}
	if err := rows.Err(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, presupuestos)
}

func upsertPresupuestoHandler(w http.ResponseWriter, r *http.Request) {
	var p Presupuesto
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		http.Error(w, "body invalido", http.StatusBadRequest)
		return
	}
	if !CategoriasValidas[p.Categoria] {
		http.Error(w, "categoria invalida", http.StatusBadRequest)
		return
	}
	if p.MontoMensual <= 0 {
		http.Error(w, "monto_mensual debe ser mayor a 0", http.StatusBadRequest)
		return
	}

	_, err := db.Exec(`
		INSERT INTO presupuestos (categoria, monto_mensual) VALUES ($1, $2)
		ON CONFLICT (categoria) DO UPDATE SET monto_mensual = EXCLUDED.monto_mensual
	`, p.Categoria, p.MontoMensual)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, p)
}

// --- Helpers ---

func presupuestoDeCategoria(categoria string) (monto float64, tiene bool, err error) {
	err = db.QueryRow(`SELECT monto_mensual FROM presupuestos WHERE categoria = $1`, categoria).Scan(&monto)
	if err == sql.ErrNoRows {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, err
	}
	return monto, true, nil
}

func totalGastadoEnElMes(categoria string, fecha time.Time) (float64, error) {
	var total sql.NullFloat64
	err := db.QueryRow(`
		SELECT SUM(monto) FROM gastos
		WHERE categoria = $1
		  AND date_trunc('month', fecha) = date_trunc('month', $2::date)
	`, categoria, fecha).Scan(&total)
	if err != nil {
		return 0, err
	}
	if !total.Valid {
		return 0, nil
	}
	return total.Float64, nil
}

func truncarADia(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}

func writeJSON(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}


nada.Test()