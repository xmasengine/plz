# plz

PLZ is a PL/M inspired compiler suite for the Z80 CPU implemented in Go language.


Differences

Although PLZ is inspired by PL/M as a high level syntax low level
effect language, and uses similar syntax, if differs in the following
points:


* PLZ uses CONSTANT, DATA as head keywords in stead of the convoluted
  PL/M DECLARE syntax. It will still use DECLARE for variable definitions.
* PLZ uses the types BYTE and WORD in stead of BYTE and ADDRESS.



# Credits

PLZ uses github.com/koron-go/z80 as the z80 emulator for testing, under the MIT
License.

The z80 assembler is based on github.com/paulhankin/z80asm and was imported
into this project for modifications under the MIT license.

Thanks to you all!



