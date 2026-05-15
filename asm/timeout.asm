		org 0
		const outp = 7
		jp start
		org 1000

		start:
		.loop
		ld a,'+'
		out (outp), a
		jp loop
		halt
