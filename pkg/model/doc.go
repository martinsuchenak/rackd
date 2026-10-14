// Package model defines Rackd's domain types: devices, networks, pools,
// addresses, users, roles and the other records the API, storage and services
// share.
//
// It is public so that extensions built on pkg/extension can use the same types
// as core. Until Rackd 1.0 it is unstable: types and fields may change in any
// release, with changes listed in the changelog.
package model
