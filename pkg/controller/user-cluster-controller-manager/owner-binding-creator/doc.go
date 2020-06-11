/*
The ownerbindingcreator controller is responsible for making sure that the binding exists and if it doesn't, will create
it and use the cluster owner as subject. It is not responsible for doing any changes to the binding once it exists, this
is done by users via the API/UI.
*/
package ownerbindingcreator
