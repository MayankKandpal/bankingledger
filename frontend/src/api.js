const BASE = 'http://localhost:8080'

export async function getAccounts() {
  const res = await fetch(`${BASE}/accounts`)
  return res.json()
}

export async function createAccount(name, mobile) {
  const res = await fetch(`${BASE}/accounts`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ name, mobile }),
  })
  return { ok: res.status === 201, data: await res.json() }
}

export async function getTransfers() {
  const res = await fetch(`${BASE}/transfers`)
  return res.json()
}

export async function createTransfer(fromAccountId, toAccountId, amount) {
  const res = await fetch(`${BASE}/transfers`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ from_account_id: fromAccountId, to_account_id: toAccountId, amount }),
  })
  return { ok: res.status === 201, data: await res.json() }
}

export async function reverseTransfer(id) {
  const res = await fetch(`${BASE}/transfers/${id}/reverse`, { method: 'POST' })
  return { ok: res.ok, data: await res.json() }
}

export async function getAuditLog() {
  const res = await fetch(`${BASE}/audit-log`)
  return res.json()
}

export async function getLedgerEntries(accountId) {
  const url = accountId
    ? `${BASE}/ledger-entries?account_id=${accountId}`
    : `${BASE}/ledger-entries`
  const res = await fetch(url)
  return res.json()
}
