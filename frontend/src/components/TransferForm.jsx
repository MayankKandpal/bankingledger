import { useState } from 'react'
import { createTransfer } from '../api'

export default function TransferForm({ accounts, onRefresh }) {
  const [fromId, setFromId] = useState('')
  const [toId, setToId] = useState('')
  const [amount, setAmount] = useState('')
  const [msg, setMsg] = useState(null)

  async function handleSubmit(e) {
    e.preventDefault()
    setMsg(null)
    const { ok, data } = await createTransfer(fromId, toId, amount)
    if (ok) {
      setAmount('')
      setMsg({ ok: true, text: `✅ Transfer completed — ref: ${data.id}` })
      onRefresh()
    } else {
      setMsg({ ok: false, text: `❌ ${data.error || 'Insufficient funds'}` })
      onRefresh() // balances unchanged but refresh to be sure
    }
  }

  return (
    <section>
      <h2>New Transfer</h2>
      <form onSubmit={handleSubmit}>
        <select value={fromId} onChange={e => setFromId(e.target.value)} required>
          <option value="">From account</option>
          {accounts.map(a => (
            <option key={a.id} value={a.id}>{a.name} (₹{a.balance})</option>
          ))}
        </select>
        <select value={toId} onChange={e => setToId(e.target.value)} required>
          <option value="">To account</option>
          {accounts.map(a => (
            <option key={a.id} value={a.id}>{a.name} (₹{a.balance})</option>
          ))}
        </select>
        <input
          type="text"
          placeholder="Amount"
          value={amount}
          onChange={e => setAmount(e.target.value)}
          required
        />
        <button type="submit">Transfer</button>
      </form>
      {msg && <p className={`msg ${msg.ok ? 'success' : 'error'}`}>{msg.text}</p>}
    </section>
  )
}
