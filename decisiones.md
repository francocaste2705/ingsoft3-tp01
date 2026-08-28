# Decisiones — TP1

## 1. Por qué Git no pudo resolver el conflicto solo

El conflicto se fabricó a propósito siguiendo la guía: dos ramas (`feature/titulo-a` y
`feature/titulo-b`) nacieron ambas desde el mismo commit de `main`, y cada una modificó
**la misma línea** del `README.md` (la primera línea, el título del proyecto) con contenido
distinto ("versión A" vs "versión B").

Git resuelve merges automáticamente cuando los cambios tocan **partes distintas** del archivo,
comparando cada rama contra el ancestro común. Pero cuando dos ramas cambian la misma línea, Git
no tiene forma de saber cuál de las dos versiones es "la correcta" — ambas son ediciones válidas
sobre el mismo punto del archivo, y elegir entre ellas es una decisión de contenido, no algo que
se pueda inferir del historial. Por eso Git se detiene y delega la decisión a una persona,
marcando el archivo con `<<<<<<<`, `=======` y `>>>>>>>`.

Para que este conflicto nunca hubiera aparecido, hubiera bastado con que las dos ramas no
tocaran la misma línea (por ejemplo, si una editaba el título y la otra agregaba una sección
distinta), o con integrar la rama B **después** de traer los cambios de A a `main` antes de
empezar a editar desde ahí — es decir, con ramas más cortas e integración más frecuente, que es
justamente lo que reduce la probabilidad y el tamaño de los conflictos en un equipo real.

## 2. Problemas encontrados y cómo los solucioné

No tuve problemas con la configuración de "Require approvals" (quedó en cero aprobaciones
como pide la guía, para poder mergear mis propios PRs al ser un trabajo individual). El resto
de los pasos —crear el repo, proteger `main`, hacer los PRs, provocar y resolver el conflicto,
y publicar el tag y la release— los hice todos de forma manual desde la web y la consola,
siguiendo la guía paso a paso, sin inconvenientes que resolver.

## 3. Declaración de uso de IA

No usé IA para realizar ninguno de los pasos técnicos del TP (creación del repositorio,
configuración de las protecciones de rama, Pull Requests, resolución del conflicto, tag o
release): todo el trabajo en el repositorio fue manual.

Sí usé IA (Claude) puntualmente al final, para redactar este archivo (`decisiones.md`) y
`evidencias.md` a partir de las capturas ya tomadas y del propio enunciado del TP. La
verificación consistió en revisar que lo escrito describiera con precisión lo que efectivamente
hice y lo que muestran mis capturas (por ejemplo, que la explicación del conflicto coincida con
las ramas y el archivo que realmente usé), y en poder explicar cada punto sin depender del texto
generado, ya que la defensa oral se responde con mis propias palabras.

## TP2 — Contenedores

### 1. Qué app elegí y por qué

Elegí armar una aplicación de **gestión de gastos personales** desde cero, con backend en **Go**
(librería estándar `net/http` + `database/sql`, sin framework) y frontend en **React + Vite**,
sobre una base de datos **PostgreSQL**.

Contra los criterios de la guía:
- **Buildea y corre localmente sin magia**: la probé end-to-end antes de dar por terminado el TP
  (carga, listado, borrado y resumen de gastos funcionando).
- **Tamaño acotado**: es un CRUD con una sola entidad (gastos) y una pantalla — alcanza para
  demostrar los tres requisitos (backend + frontend + base de datos) sin fricción de más.
- **La entiendo por completo**: la escribí yo mismo (con asistencia de IA, ver declaración más
  abajo), así que puedo modificarla y explicar cada parte en la defensa oral.
- Elegí Go en vez de un stack más "tradicional" de la materia porque me interesaba probar un
  lenguaje que compila a un binario único — eso simplifica notablemente el Dockerfile del backend
  (no hace falta instalar ningún runtime en la imagen final, alcanza con copiar el binario sobre
  una base mínima como `alpine`).

### 2. Decisiones de contenerización

- **Imágenes base**: `golang:1.22-alpine` para compilar el backend, `alpine:3.20` para ejecutarlo
  en producción; `node:22-alpine` para compilar el frontend, `nginx:alpine` para servirlo.
- **Estructura multi-stage**: en las dos imágenes, la etapa de build (con el compilador/SDK) se
  descarta por completo en la imagen final — solo viaja el binario de Go compilado, o los
  estáticos que generó `vite build`.
- **Qué persiste y qué no**: los datos de PostgreSQL viven en el volumen nombrado `db_data`,
  montado en `/var/lib/postgresql/data`. Los contenedores de `backend` y `frontend` son
  completamente efímeros — no guardan ningún estado propio, así que se pueden recrear sin
  pérdida de información.
- **Comunicación entre servicios**: el backend se conecta a la base por el nombre de servicio
  `db` (`DATABASE_URL=postgres://postgres:${DB_PASSWORD}@db:5432/app`), y el frontend habla con
  el backend por ruta relativa (`/api/...`), que nginx reenvía internamente a `http://backend:8080`
  — así el navegador nunca necesita conocer el nombre `backend` (que no resolvería fuera de la
  red de compose).
- **Secretos**: la contraseña de la base viaja por la variable `DB_PASSWORD`, definida en un
  `.env` que no se commitea (con `.env.example` como plantilla commiteada).
- **Registry**: publiqué las dos imágenes en GitHub Container Registry (ghcr.io), con tag
  `v0.1.0`, y las hice públicas para que `docker-compose.registry.yml` pueda bajarlas sin
  credenciales.

### 3. Problemas encontrados y cómo los resolví

- **`go mod download` no generaba un `go.sum` completo**: el build fallaba con
  `missing go.sum entry for module providing package github.com/lib/pq`. Lo resolví copiando
  todo el código fuente antes de correr `go mod tidy` (en vez de `go mod download` con solo el
  `go.mod`), porque `tidy` sí escanea los imports reales y arma el `go.sum` correcto.
- **`npm ci` fallaba** porque no tenía un `package-lock.json` commiteado (lo esperable con
  `npm ci`, que exige un lockfile ya generado). Lo resolví usando `npm install` en el Dockerfile,
  que genera el lockfile sobre la marcha si no existe.
- **El primer `docker push` del backend falló** con `error from registry: unknown` después de
  subir todas las capas — fue un error transitorio del lado de GitHub (probablemente por ser la
  primera vez que se creaba ese paquete en mi cuenta). Reintentar el mismo comando lo resolvió sin
  cambiar nada.

### 4. Declaración de uso de IA

Usé IA (Claude) de forma extensa en este TP, a diferencia del TP1: le pedí que me ayudara a
elegir el stack (Go + React) y que armara la aplicación completa de gestión de gastos —
backend, frontend, Dockerfiles multi-stage, `docker-compose.yml`, `docker-compose.registry.yml`
y el `nginx.conf`. Después corrí yo mismo cada comando de build, arranque y prueba, y cuando
aparecieron errores (los tres del punto anterior) se los reporté a la IA, que me indicó la causa
y el cambio puntual a hacer en el Dockerfile correspondiente.

Verifiqué lo generado ejecutándolo yo mismo en mi máquina en cada paso — no di nada por
funcionando hasta ver la salida real de mis propios comandos (`docker compose ps`, `curl`,
la interfaz en el navegador, los `docker images` de la comparación de tamaños, y el `pull`
deslogueado para confirmar que las imágenes eran realmente públicas). Entiendo la estructura de
cada Dockerfile y del compose lo suficiente como para explicarla en la defensa oral: qué hace
cada instrucción, por qué el orden importa para el cache, por qué el volumen es necesario, y por
qué el frontend habla con el backend por ruta relativa y no por URL absoluta.
