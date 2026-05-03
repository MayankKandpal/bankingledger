import { useState } from 'react'
import { createAccount } from '../api'

export default function AccountList({ accounts, onRefresh }) {
  const [name, setName] = useState('')
  const [mobile, setMobile] = useState('')
  const [msg, setMsg] = useState(null)

  async function handleCreate(e) {
    e.preventDefault()
    setMsg(null)
    if (!name.trim()) {
      setMsg({ ok: false, text: 'Name is required' })
      return
    }
    const { ok, data } = await createAccount(name.trim(), mobile.trim())
    if (!ok) {
      setMsg({ ok: false, text: data.error || 'Failed to create account' })
      return
    }
    setName('')
    setMobile('')
    setMsg({ ok: true, text: `Account "${data.name}" created` })
    onRefresh()
  }

  return (
    <section>
      <h2>Accounts</h2>

      {accounts.length === 0 ? (
        <p className="empty">No accounts yet.</p>
      ) : (
        <table>
          <thead>
            <tr>
              <th>Name</th>
              <th>Mobile</th>
              <th>Balance</th>
            </tr>
          </thead>
          <tbody>
            {accounts.map(a => (
              <tr key={a.id}>
                <td>{a.name}</td>
                <td>{a.mobile || '—'}</td>
                <td>₹{a.balance}</td>
              </tr>
            ))}
          </tbody>
        </table>
      )}

      <h3>Create Account</h3>
      <form onSubmit={handleCreate}>
        <input
          placeholder="Name"
          value={name}
          onChange={e => setName(e.target.value)}
        />
        <input
          placeholder="Mobile (optional)"
          value={mobile}
          onChange={e => setMobile(e.target.value)}
        />
        <button type="submit">Create</button>
      </form>
      {msg && <p className={`msg ${msg.ok ? 'success' : 'error'}`}>{msg.text}</p>}
    </section>
  )
}
