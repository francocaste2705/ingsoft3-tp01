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
