// Buv is a web server dedicated to being a slave to Web 2.0, serving up web pages,
// parsing templates and providing a logger to client handling code. It allows
// a client to specify gorilla-style mux's to specific handlers, and exposes the gorilla
// session interface to handlers using secure cookies. Furthermore, redirection
// code can be specified separately from handler code, keeping content-delivery interests
// orthogonal to authentication-authorization-access interests.
package buv

/*
	This file is a part of Buv
	Copyright (C) 2014  Cory J. Slep

    Buv is free software: you can redistribute it and/or modify
    it under the terms of the GNU Lesser General Public License as published by
    the Free Software Foundation, either version 3 of the License, or
    (at your option) any later version.

    Buv is distributed in the hope that it will be useful,
    but WITHOUT ANY WARRANTY; without even the implied warranty of
    MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
    GNU Lesser General Public License for more details.

    You should have received a copy of the GNU Lesser General Public License
    along with this program.  If not, see <http://www.gnu.org/licenses/>.
*/
