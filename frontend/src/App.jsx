import { useEffect, useState } from 'react'
// Nota: agregar a App.css la clase .excedido { color: #f87171; font-weight: bold; }

const CATEGORIAS = ['Comida', 'Transporte', 'Ocio', 'Servicios', 'Salud', 'Otros']

function hoyISO() {
  return new Date().toISOString().slice(0, 10)
}

export default function App() {
  const [gastos, setGastos] = useState([])
  const [resumen, setResumen] = useState([])
  const [presupuestos, setPresupuestos] = useState([])

  const [descripcion, setDescripcion] = useState('')
  const [monto, setMonto] = useState('')
  const [categoria, setCategoria] = useState(CATEGORIAS[0])
  const [fecha, setFecha] = useState(hoyISO())

  const [presupuestoCategoria, setPresupuestoCategoria] = useState(CATEGORIAS[0])
  const [presupuestoMonto, setPresupuestoMonto] = useState('')

  const [error, setError] = useState('')
  const [cargando, setCargando] = useState(true)

  async function cargar() {
    try {
      const [gastosRes, resumenRes, presupuestosRes] = await Promise.all([
        fetch('/api/gastos'),
        fetch('/api/resumen'),
        fetch('/api/presupuestos'),
      ])
      setGastos(await gastosRes.json())
      setResumen(await resumenRes.json())
      setPresupuestos(await presupuestosRes.json())
    } finally {
      setCargando(false)
    }
  }

  useEffect(() => {
    cargar()
  }, [])

  // El formulario es invalido si falta algun dato o la fecha es futura.
  // Con esto deshabilitamos el boton en vez de dejar que el usuario
  // descubra el error recien despues de enviarlo.
  const formularioValido =
    descripcion.trim() !== '' &&
    Number(monto) > 0 &&
    fecha !== '' &&
    fecha <= hoyISO()

  async function agregarGasto(e) {
    e.preventDefault()
    setError('')
    const res = await fetch('/api/gastos', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        descripcion,
        monto: parseFloat(monto),
        categoria,
        fecha: new Date(fecha).toISOString(),
      }),
    })
    if (!res.ok) {
      const texto = await res.text()
      // El backend devuelve 422 puntualmente cuando se excede el
      // presupuesto de la categoria: se lo mostramos distinto al resto
      // de los errores de validacion.
      if (res.status === 422) {
        setError(`Presupuesto excedido: ${texto}`)
      } else {
        setError(texto)
      }
      return
    }
    setDescripcion('')
    setMonto('')
    cargar()
  }

  async function eliminarGasto(id) {
    await fetch(`/api/gastos/${id}`, { method: 'DELETE' })
    cargar()
  }

  async function marcarComoPagado(id) {
    await fetch(`/api/gastos/${id}/estado`, {
      method: 'PATCH',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ estado: 'pagado' }),
    })
    cargar()
  }

  async function guardarPresupuesto(e) {
    e.preventDefault()
    setError('')
    const res = await fetch('/api/presupuestos', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        categoria: presupuestoCategoria,
        monto_mensual: parseFloat(presupuestoMonto),
      }),
    })
    if (!res.ok) {
      setError(await res.text())
      return
    }
    setPresupuestoMonto('')
    cargar()
  }

  const total = gastos.reduce((acc, g) => acc + Number(g.monto), 0)

  return (
    <div className="container">
      <h1>Gestión de Gastos Personales</h1>

      <form onSubmit={agregarGasto} className="form">
        <input
          placeholder="Descripción"
          value={descripcion}
          onChange={(e) => setDescripcion(e.target.value)}
          required
        />
        <input
          type="number"
          step="0.01"
          min="0.01"
          placeholder="Monto"
          value={monto}
          onChange={(e) => setMonto(e.target.value)}
          required
        />
        <select value={categoria} onChange={(e) => setCategoria(e.target.value)}>
          {CATEGORIAS.map((c) => (
            <option key={c} value={c}>
              {c}
            </option>
          ))}
        </select>
        <input
          type="date"
          value={fecha}
          max={hoyISO()}
          onChange={(e) => setFecha(e.target.value)}
        />
        <button type="submit" disabled={!formularioValido}>
          Agregar
        </button>
      </form>

      {error && <p className="error">{error}</p>}

      <details className="presupuestos">
        <summary>Presupuestos mensuales por categoría</summary>
        <form onSubmit={guardarPresupuesto} className="form">
          <select
            value={presupuestoCategoria}
            onChange={(e) => setPresupuestoCategoria(e.target.value)}
          >
            {CATEGORIAS.map((c) => (
              <option key={c} value={c}>
                {c}
              </option>
            ))}
          </select>
          <input
            type="number"
            step="0.01"
            min="0.01"
            placeholder="Monto mensual"
            value={presupuestoMonto}
            onChange={(e) => setPresupuestoMonto(e.target.value)}
            required
          />
          <button type="submit" disabled={!(Number(presupuestoMonto) > 0)}>
            Guardar presupuesto
          </button>
        </form>
        <ul>
          {presupuestos.map((p) => (
            <li key={p.categoria}>
              {p.categoria}: ${Number(p.monto_mensual).toFixed(2)} / mes
            </li>
          ))}
        </ul>
      </details>

      {cargando ? (
        <p>Cargando...</p>
      ) : (
        <>
          <h2>Total: ${total.toFixed(2)}</h2>

          <div className="resumen">
            <h3>Por categoría</h3>
            <ul>
              {resumen.map((r) => (
                <li key={r.categoria}>
                  {r.categoria}: ${Number(r.total).toFixed(2)}
                  {r.porcentaje_usado != null && (
                    <span className={r.porcentaje_usado > 100 ? 'excedido' : ''}>
                      {' '}
                      ({r.porcentaje_usado.toFixed(0)}% del presupuesto)
                    </span>
                  )}
                </li>
              ))}
            </ul>
          </div>

          <table>
            <thead>
              <tr>
                <th>Fecha</th>
                <th>Descripción</th>
                <th>Categoría</th>
                <th>Monto</th>
                <th>Estado</th>
                <th></th>
              </tr>
            </thead>
            <tbody>
              {gastos.map((g) => (
                <tr key={g.id}>
                  <td>{new Date(g.fecha).toLocaleDateString()}</td>
                  <td>{g.descripcion}</td>
                  <td>{g.categoria}</td>
                  <td>${Number(g.monto).toFixed(2)}</td>
                  <td>{g.estado}</td>
                  <td>
                    <button
                      onClick={() => marcarComoPagado(g.id)}
                      disabled={g.estado === 'pagado'}
                    >
                      Marcar pagado
                    </button>
                    <button onClick={() => eliminarGasto(g.id)}>Eliminar</button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </>
      )}
    </div>
  )
}
