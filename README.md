Buv Web Server
==============

Table of Contents
-----------------

1. What Is Buv?
2. Features
3. How To
4. License

What Is Buv? (Baby Don't Hurt Me!)
----------------------------------

Buv is a configurable web server designed to abstract away the mechanics of
serving HTTP and AJAX requests while allowing client code to still be able
to have full support for standard web features. These features are manifested
in Buv by the [gorilla web toolkit](http://www.gorillatoolkit.org/).

Features
--------

Buv is configurable to allow clients to:

* Specify port and domain to service
* Use [gorilla-mux](http://www.gorillatoolkit.org/pkg/mux)-style pattern matching to designate request handlers based on:
	* Schemes
	* URI
	* HTTP method
	* Queries
	* A parent's patterns
* Register handlers guarded by redirecting functions
* Create, rotate, and configure secure cookies
* Specify secure cookie lifetimes & whether to only modify cookies over HTTP
* Save Buv's configuration to file for easier instantiation
* Specify a default handler for nonexistant resources
* Register template files for handler use
	* Notifies client if not all correct template dependencies are added (*no manual testing of every template needed*)
* Drop favicon.ico at the root
* Designate special folders to serve assets from
* Gracefully terminate open connections upon shutdown

The handler-specific benefits include:

* Web logging services
* Session value setting, retrieving, and erasing
* Session flash setting and retrieving
* HTTP method of the request
* Manual redirection to another URI with an HTTP status code
* Access to the URL of the request
* Any query and post values of the request
* Render templates that are registered with the server
* Fetch another valid URL for another URI

How To
------

**Todo!**

License
-------

This work is released under the [GNU's Lesser General Public License version 3 (LGPLv3)](http://www.gnu.org/copyleft/lesser.html).

Please see the Copying and Copying.Lesser files for the license body.