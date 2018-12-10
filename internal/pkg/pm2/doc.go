/*
Package pm2 facilitates starting external
processes on which our API depends.

For instance, it starts Chrome headless and
unoconv listener with PM2.

The PM2 process manager launch those processes and keep
them running in the background. If for some reason they
crash, it will also restart them.
*/
package pm2
