	org 0
	const outp = 7
	const OUT_PORT = 7
	jp start

	hello:
	ld a,'H'
	out (OUT_PORT), a
	ld a, 'E'
	out (outp), a
	ld a, 'L'
	out (outp), a
	ld a, 'L'
	out (outp), a
	ld a, 'O'
	out (outp), a
	ret

	start:
	call hello
	halt
