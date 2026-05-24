# plz

PLZ is a PL/M inspired compiler suite for the Z80 CPU implemented in Go language.
PL/M is a high level syntax low level effect language developed in the
1970-1980 era by Kendall at Intel.

While C is said to be a low level language, it is not. The C language abstracts
the machine and has no direct support for low level features like interrupts,
chip level i/o, locating code in ROM, or memory mapping. With C we have to rely
on undefined behavior of the language or on assembly.

In PLZ there should be no undefined behavior.  Statements like INPUT, OUTPUT and
INTERRUPT allow true low level programming without assembly. The syntax,
while somewhat verbose, uses easy to read full English keywords.
This resembles of BASIC or PL/1, without being as verbose as COBOL, or FORTRAN.


# Differences

Although PLZ is inspired by PL/M as a high level syntax low level
effect language, and uses similar syntax, if differs or is planned to
differ in the following points:


* PLZ requires a LET keyword for assignments to simplify the parser.
* PLZ uses CONSTANT, DATA as head keywords in stead of the convoluted
  PL/M DECLARE syntax. It will still use DECLARE for variable definitions.
* PLM only allows to declare one variable, constant or data per statement to
  simplify the parser.
* PLZ uses the types BYTE and WORD in stead of BYTE and ADDRESS.
* PLZ, like Go language, does not require ; separators between statements
  and instead decides on the end of statement either using keywords
  or newline characters. Nevertheless ; will be allowed as a statement
  separator as well.
* PLZ has a SWITCH statement with proper CASE blocks and default handling,
  unlike the PL/M case.
* PLZ only allows one named label and one numbered label per statement.



# Planned features

At first the target is to implement most features of PL/M 73.
We since the Z80 also has interrupts, we will also include features from
PL/M 80 to support those. Apart from these features, and inspired by BASIC
and other languages,  I also would like to add support for the following
features:

* TASK: a light weight cooperative task system, like the APOLLO guidance
  computer or Seiken Densetsu 2 used. Like this we can program without state
  machines. There will be up to 16 statically allocated named tasks which can
  SLEEP, SUSPEND, RESUME, etc.
* BANK: Handling of banked ROM.
* SAVE: Handling of battery backed RAM.
* MUSIC: support for game music, probably based on the TASK system.
* SOUND: support for game sounds, probably based on the TASK system.
* SCREEN, TILE, SPRITE: screen and video handling.


# Credits

PLZ uses github.com/koron-go/z80 as the z80 emulator for testing, under the MIT
License.

The z80 assembler is based on github.com/paulhankin/z80asm and was imported
into this project for modifications under the MIT license.

Thanks to you all!

# License

MIT License

Copyright (c) 2026 xmasengine

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.



