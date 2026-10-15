# Laravel legacy

- `legacy` loads Laravel-era DB once, on staging copy. Rarely touched. No new Go-era logic.
- `legacy/restore` converts, saves only upgraded file as ordinary backup. Live data changes only when user restores it.
- Drives only `store`. HTTP may import `legacy/restore`, never `legacy`. `backup` and `ledger` know nothing of Laravel.
