import { useState, useEffect } from 'react'
import { getLedgerEntries } from '../api'

const PAGE_SIZE = 10

export default function LedgerEntries({ accounts }) {
  const [accountId, setAccountId] = useState('')
  const [entries, setEntries] = useState([])
  const [page, setPage] = useState(0)

  useEffect(() => {
    if (!accountId) {
      setEntries([])
      setPage(0)
      return
    }
    getLedgerEntries(accountId).then(data => {
      setEntries(data || [])
      setPage(0)
    })
  }, [accountId])

  const totalPages = Math.max(1, Math.ceil(entries.length / PAGE_SIZE))
  const shown = entries.slice(page * PAGE_SIZE, (page + 1) * PAGE_SIZE)

  const accountName = id => {
    if (!id) return '—'
    if (id === '00000000-0000-0000-0000-000000000000') return 'SYSTEM'
    return accounts.find(a => a.id === id)?.name ?? id.slice(0, 8) + '…'
  }

  return (
    <section>
      <h2>Ledger Entries</h2>
      <div className="filter-bar">
        <select value={accountId} onChange={e => setAccountId(e.target.value)}>
          <option value="">Select account to view ledger</option>
          {accounts.map(a => (
            <option key={a.id} value={a.id}>{a.name}</option>
          ))}
        </select>
        {entries.length > 0 && (
          <span className="count">{entries.length} entries</span>
        )}
      </div>
      {!accountId ? null : entries.length === 0 ? (
        <p className="empty">No ledger entries for this account.</p>
      ) : (
        <>
          <table>
            <thead>
              <tr>
                <th>Time</th>
                <th>Transfer Ref</th>
                <th>Account</th>
                <th>Amount</th>
              </tr>
            </thead>
            <tbody>
              {shown.map(e => (
                <tr key={e.id}>
                  <td>{new Date(e.created_at).toLocaleTimeString()}</td>
                  <td title={e.transfer_id}>{e.transfer_id.slice(0, 8)}…</td>
                  <td>{accountName(e.account_id)}</td>
                  <td className={parseFloat(e.amount) >= 0 ? 'credit' : 'debit'}>
                    {parseFloat(e.amount) >= 0 ? '+' : ''}₹{e.amount}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
          {totalPages > 1 && (
            <div className="pagination">
              <button onClick={() => setPage(p => p - 1)} disabled={page === 0}>← Prev</button>
              <span>Page {page + 1} of {totalPages}</span>
              <button onClick={() => setPage(p => p + 1)} disabled={page >= totalPages - 1}>Next →</button>
            </div>
          )}
        </>
      )}
    </section>
  )
}
