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

## TP3 — Planificación y trazabilidad

### 1. Duración del sprint

Elegí sprints de **1 semana**, porque coincide con el ritmo de entrega de la cátedra (un TP
por semana) y permite cerrar un ciclo completo de planificación → ejecución → revisión en cada
entrega, alineando el objetivo del sprint con el objetivo de cada práctico. Un sprint más largo
(por ejemplo, un mes) no me daría feedback a tiempo respecto del calendario real de la materia.

### 2. Límite de trabajo en progreso

Configuré el límite de la columna *In Progress* en **2**, siguiendo la regla de arranque de la
guía: cantidad de personas trabajando (1, en mi caso, porque el TP es individual) más uno. El
"+1" es la válvula para cuando algo queda esperando (por ejemplo, una revisión propia o un
bloqueo externo) y necesito poder avanzar en otra cosa sin romper el límite. Si en la práctica
nunca llego a usar las 2 posiciones, es señal de que podría bajarlo a 1; si lo alcanzo todo el
tiempo y me frena, sería señal de subirlo.

### 3. Diagnóstico de la historia mal escrita

La historia de ejemplo *"Como desarrollador quiero crear la tabla usuarios"* está mal escrita
porque es una **tarea técnica disfrazada de historia**: no describe una capacidad observable
por un usuario, sino un paso de implementación interno. Le falta además el "para qué" (el
beneficio) — "crear una tabla" no es algo que nadie "quiera" en sí mismo, es un medio para
lograr otra cosa. Se puede reconocer por el anti-patrón que menciona la guía: si el "quiero"
describe una pieza técnica en vez de una capacidad de negocio, es una tarea.

Cómo la reescribiría: la elevaría a una historia real, por ejemplo *"Como usuario quiero
registrarme e iniciar sesión para poder acceder a mis datos personales"*, y "crear la tabla
usuarios" pasaría a ser una de las **tareas técnicas** dentro de esa historia (junto con, por
ejemplo, "implementar el endpoint de registro" o "validar el formato del email").

### 4. Problemas encontrados y cómo los resolví

- **Los comandos `gh` con `\` para continuar en varias líneas fallaban en PowerShell**
  (`ParserError: Missing expression after unary operator '--'`). PowerShell no interpreta la
  barra invertida como continuación de línea (eso es sintaxis de bash/zsh). Lo resolví escribiendo
  cada comando `gh issue create` en una sola línea, con comillas dobles en vez de simples.
- **El primer `gh project item-add` falló** porque copié el número de proyecto entre signos
  `< >` literales (`<2>`), que PowerShell interpretó como redirección de archivo. Lo resolví
  usando el número solo, sin los signos.
- **El PR que debía cerrar la tarea "Escribir el workflow de build y tests" (#9) se mergeó con
  `Closes #12`** — un número de issue que no existe en mi repositorio (probablemente copiado de
  un ejemplo sin ajustar). Como el merge ya había ocurrido, el auto-cierre no se disparó
  retroactivamente. Lo resolví abriendo un segundo PR chico (un comentario adicional en el
  `ci.yml`) con la descripción corregida a `Closes #9`, que sí cerró la tarea correcta al
  mergearse.

### 5. Declaración de uso de IA

Usé IA (Claude) para traducir la guía del TP a una secuencia concreta de comandos `gh` y pasos
de la interfaz web adaptados a mi repositorio, y para diagnosticar los tres problemas del punto
anterior a medida que aparecían (en particular, para identificar por qué `Closes #12` no había
cerrado ningún issue). Verifiqué cada paso ejecutándolo yo mismo y revisando el resultado real
en GitHub: el estado de cada issue con `gh issue list`, la jerarquía de sub-issues abriendo cada
uno en la web, el tablero y el límite de trabajo en progreso, y finalmente que la tarea #9
apareciera cerrada y enlazada al PR correcto después de corregir el error. Entiendo la
diferencia entre historia, tarea y épica, por qué la trazabilidad depende de que `Closes #N`
esté en la descripción del PR y no en un comentario posterior, y por qué elegí la duración de
sprint y el límite de trabajo en progreso que elegí — puedo explicar cada uno de estos puntos
en la defensa oral sin depender del texto generado.
