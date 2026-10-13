# 02 Remove the automatic personal space

A user can have zero spaces. Guests already do, and the SPA already handles it: `SpaceGate` shows `NoSpace` (with a guest variant).

## Work

- Remove the three `spaces.Provision` calls: `authRegister`, `usersStore` ([users.go](../../internal/httpserver/users.go)) and the SSO `ProvisionSpace` callback ([server.go](../../internal/httpserver/server.go)). Remove `Provision` itself and the `ProvisionSpace` field of `sso.Service`.
- `NoSpace` keeps its one-click create until task 07 replaces it.
- Check every place that assumed a new user has a space (the existing tests of registration, admin-created users and SSO).

## Tests (first)

- Registration, admin-created users and SSO-provisioned users end with zero spaces and can log in.
- `NoSpace` for a user (create works) and for a guest (message only).

## Done when

No new user gets a space, and the app works with zero spaces. `docs/spaces.md` is updated in task 09.
