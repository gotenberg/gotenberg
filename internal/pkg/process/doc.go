/*
Package process facilitates starting external
programs on which our API depends.

For instance, it starts Chrome headless and
unoconv listener with PM2.

The PM2 process manager launch those programs and keep
them running in the background. If for some reason they
crash, it will also restart them.

Note that after starting an external program, a sleep of 4
seconds is done to make sure that the program is actually
running.
*/
package process
