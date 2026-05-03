import { useState, useMemo } from 'react'
import { reverseTransfer } from '../api'

const PAGE_SIZE = 10

export default function TransferList({ transfers, accounts, onRefresh }) {
  const [page, setPage] = useState(0)
  const [filterAccountId, setFilterAccountId] = useState('')

  const accountName = id => {
    if (!id) return '—'
    if (id === '00000000-0000-0000-0000-000000000000') return 'SYSTEM'
    return accounts.find(a => a.id === id)?.name ?? id.slice(0, 8) + '…'
  }

  const filtered = useMemo(() => {
    if (!filterAccountId) return transfers
    return transfers.filter(t =>
      t.from_account_id === filterAccountId || t.to_account_id === filterAccountId
    )
  }, [transfers, filterAccountId])

  const totalPages = Math.max(1, Math.ceil(filtered.length / PAGE_SIZE))
  const shown = filtered.slice(page * PAGE_SIZE, (page + 1) * PAGE_SIZE)

  function handleFilterChange(e) {
    setFilterAccountId(e.target.value)
    setPage(0)
  }

  async function handleReverse(id) {
    await reverseTransfer(id)
    onRefresh()
  }

  function Badge({ status }) {
    const cls = {
      COMPLETED: 'badge badge-completed',
      FAILED:    'badge badge-failed',
      REVERSED:  'badge badge-reversed',
    }
    return <span className={cls[status] ?? 'badge'}>{status}</span>
  }

  return (
    <section>
      <h2>Transfers</h2>
      <div className="filter-bar">
        <select value={filterAccountId} onChange={handleFilterChange}>
          <option value="">All accounts</option>
          {accounts.map(a => (
            <option key={a.id} value={a.id}>{a.name}</option>
          ))}
        </select>
        <span className="count">{filtered.length} record{filtered.length !== 1 ? 's' : ''}</span>
      </div>
      {filtered.length === 0 ? (
        <p className="empty">No transfers found.</p>
      ) : (
        <>
          <table>
            <thead>
              <tr>
                <th>Reference</th>
                <th>From</th>
                <th>To</th>
                <th>Amount</th>
                <th>Status</th>
                <th>Action</th>
              </tr>
            </thead>
            <tbody>
              {shown.map(t => (
                <tr key={t.id}>
                  <td title={t.id}>{t.id.slice(0, 8)}…</td>
                  <td>{accountName(t.from_account_id)}</td>
                  <td>{accountName(t.to_account_id)}</td>
                  <td>₹{t.amount}</td>
                  <td><Badge status={t.status} /></td>
                  <td>
                    {t.status === 'COMPLETED' && !t.reversed_by && (
                      <button className="reverse" onClick={() => handleReverse(t.id)}>
                        Reverse
                      </button>
                    )}
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
