# subtext


Packages:

  * `parse` reads input and produces a syntax tree.
  * `macro` defines macros that transform text.
  * `render` reads the syntax tree and produces a document according to
    rendering rules.
  * `document` writes the document to a file.
  * `bundle` manages the production of a document collection such as a web
    site.

    macros are the defined things that use templates, commands are what are
    issued on the command line. Directive

  ```
  go build && echo "a\n\nb" |./subtext make test -v --logalltags
  
  ```