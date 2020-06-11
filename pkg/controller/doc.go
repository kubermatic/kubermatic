/*
Package controller contains all our controllers. They are sorted by binary they run in, which means that
for all folders here a corresponding folder in the `cmd/` directory has to exist.

The only exception here are the `util` package which does not contain any controllers but some helpers
and the `shared` package which contains controllers that run within more than one binary.
*/
package controller
