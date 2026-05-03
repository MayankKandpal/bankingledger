import { useState, useMemo } from 'react'

const PAGE_SIZE = 10

export default function AuditLog({ entries, accounts }) {
  const [page, setPage] = useState(0)
  const [filterAccountId, setFilterAccountId] = useState('')

  const accountName = id => {
    if (!id) return '—'
    if (id === '00000000-0000-0000-0000-000000000000') return 'SYSTEM'
    return accounts.find(a => a.id === id)?.name ?? id.slice(0, 8) + '…'
  }

  const filtered = useMemo(() => {
    if (!filterAccountId) return entries
    return entries.filter(e =>
      e.from_account_id === filterAccountId || e.to_account_id === filterAccountId
    )
  }, [entries, filterAccountId])

  const totalPages = Math.max(1, Math.ceil(filtered.length / PAGE_SIZE))
  const shown = filtered.slice(page * PAGE_SIZE, (page + 1) * PAGE_SIZE)

  function handleFilterChange(e) {
    setFilterAccountId(e.target.value)
    setPage(0)
  }

  function OutcomeBadge({ outcome }) {
    return (
      <span className={`badge ${outcome === 'SUCCESS' ? 'badge-completed' : 'badge-failed'}`}>
        {outcome}
      </span>
    )
  }

  return (
    <section>
      <h2>Audit Log</h2>
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
        <p className="empty">No audit entries found.</p>
      ) : (
        <>
          <table>
            <thead>
              <tr>
                <th>Time</th>
                <th>Operation</th>
                <th>From</th>
                <th>To</th>
                <th>Amount</th>
                <th>Outcome</th>
                <th>Reason</th>
              </tr>
            </thead>
            <tbody>
              {shown.map(e => (
                <tr key={e.id}>
                  <td>{new Date(e.created_at).toLocaleTimeString()}</td>
                  <td>{e.operation}</td>
                  <td>{accountName(e.from_account_id)}</td>
                  <td>{accountName(e.to_account_id)}</td>
                  <td>{e.amount ? `₹${e.amount}` : '—'}</td>
                  <td><OutcomeBadge outcome={e.outcome} /></td>
                  <td>{e.failure_reason || '—'}</td>
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
