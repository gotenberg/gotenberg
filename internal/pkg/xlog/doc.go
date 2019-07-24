/*
Package xlog defines a standard logger
for the application.

It uses structured logging thanks to
https://github.com/sirupsen/logrus.

All messages have at least two fields:

A "trace" field which helps to identify
messages belonging to the same context.

An "op" field which helps to identify
the logical operation associated
with the message.
*/
package xlog
