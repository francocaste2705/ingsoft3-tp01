import { useEffect, useState } from 'react'

const CATEGORIAS = ['Comida', 'Transporte', 'Ocio', 'Servicios', 'Salud', 'Otros']

export default function App() {
  const [gastos, setGastos] = useState([])
  const [resumen, setResumen] = useState([])
  const [descripcion, setDescripcion] = useState('')
  const [monto, setMonto] = useState('')
  const [categoria, setCategoria] = useState(CATEGORIAS[0])
  const [error, setError] = useState('')
  const [cargando, setCargando] = useState(true)

  async function cargar() {
    try {
      const [gastosRes, resumenRes] = await Promise.all([
        fetch('/api/gastos'),
        fetch('/api/resumen'),
      ])
      setGastos(await gastosRes.json())
      setResumen(await resumenRes.json())
    } finally {
      setCargando(false)
    }
  }

  useEffect(() => {
    cargar()
  }, [])

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
      }),
    })
    if (!res.ok) {
      setError(await res.text())
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
        <button type="submit">Agregar</button>
      </form>

      {error && <p className="error">{error}</p>}

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
                  <td>
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
