import { useState } from 'react'
import { createTransfer } from '../api'

const SYSTEM = '00000000-0000-0000-0000-000000000000'

export default function DepositForm({ accounts, onRefresh }) {
  const [type, setType] = useState('deposit')
  const [accountId, setAccountId] = useState('')
  const [amount, setAmount] = useState('')
  const [msg, setMsg] = useState(null)

  async function handleSubmit(e) {
    e.preventDefault()
    setMsg(null)
    const fromId = type === 'deposit' ? SYSTEM : accountId
    const toId   = type === 'deposit' ? accountId : SYSTEM
    const { ok, data } = await createTransfer(fromId, toId, amount)
    if (ok) {
      setAmount('')
      setMsg({ ok: true, text: `✅ ₹${amount} ${type === 'deposit' ? 'deposited into' : 'withdrawn from'} ${accounts.find(a => a.id === accountId)?.name}` })
      onRefresh()
    } else {
      setMsg({ ok: false, text: `❌ ${data.error || 'Transaction failed'}` })
    }
  }

  return (
    <section>
      <h2>Deposit / Withdraw</h2>
      <form onSubmit={handleSubmit}>
        <select value={type} onChange={e => { setType(e.target.value); setMsg(null) }}>
          <option value="deposit">Deposit</option>
          <option value="withdraw">Withdraw</option>
        </select>
        <select value={accountId} onChange={e => setAccountId(e.target.value)} required>
          <option value="">Select account</option>
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
        <button type="submit">{type === 'deposit' ? 'Deposit' : 'Withdraw'}</button>
      </form>
      {msg && <p className={`msg ${msg.ok ? 'success' : 'error'}`}>{msg.text}</p>}
    </section>
  )
}
