/*
Package xexec helps creating exec.Cmd
with logging and executing those commands
without leaking orphan processes.

All functions return our standard xerror.Error
in case of error.
*/
package xexec
