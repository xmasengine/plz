package isa

const (
	RLC_B       BitOpcode = 0
	RLC_C       BitOpcode = 1
	RLC_D       BitOpcode = 2
	RLC_E       BitOpcode = 3
	RLC_H       BitOpcode = 4
	RLC_L       BitOpcode = 5
	RLC_PtrHL   BitOpcode = 6
	RLC_A       BitOpcode = 7
	RRC_B       BitOpcode = 8
	RRC_C       BitOpcode = 9
	RRC_D       BitOpcode = 10
	RRC_E       BitOpcode = 11
	RRC_H       BitOpcode = 12
	RRC_L       BitOpcode = 13
	RRC_PtrHL   BitOpcode = 14
	RRC_A       BitOpcode = 15
	RL_B        BitOpcode = 16
	RL_C        BitOpcode = 17
	RL_D        BitOpcode = 18
	RL_E        BitOpcode = 19
	RL_H        BitOpcode = 20
	RL_L        BitOpcode = 21
	RL_PtrHL    BitOpcode = 22
	RL_A        BitOpcode = 23
	RR_B        BitOpcode = 24
	RR_C        BitOpcode = 25
	RR_D        BitOpcode = 26
	RR_E        BitOpcode = 27
	RR_H        BitOpcode = 28
	RR_L        BitOpcode = 29
	RR_PtrHL    BitOpcode = 30
	RR_A        BitOpcode = 31
	SLA_B       BitOpcode = 32
	SLA_C       BitOpcode = 33
	SLA_D       BitOpcode = 34
	SLA_E       BitOpcode = 35
	SLA_H       BitOpcode = 36
	SLA_L       BitOpcode = 37
	SLA_PtrHL   BitOpcode = 38
	SLA_A       BitOpcode = 39
	SRA_B       BitOpcode = 40
	SRA_C       BitOpcode = 41
	SRA_D       BitOpcode = 42
	SRA_E       BitOpcode = 43
	SRA_H       BitOpcode = 44
	SRA_L       BitOpcode = 45
	SRA_PtrHL   BitOpcode = 46
	SRA_A       BitOpcode = 47
	SLL_B       BitOpcode = 48
	SLL_C       BitOpcode = 49
	SLL_D       BitOpcode = 50
	SLL_E       BitOpcode = 51
	SLL_H       BitOpcode = 52
	SLL_L       BitOpcode = 53
	SLL_PtrHL   BitOpcode = 54
	SLL_A       BitOpcode = 55
	SRL_B       BitOpcode = 56
	SRL_C       BitOpcode = 57
	SRL_D       BitOpcode = 58
	SRL_E       BitOpcode = 59
	SRL_H       BitOpcode = 60
	SRL_L       BitOpcode = 61
	SRL_PtrHL   BitOpcode = 62
	SRL_A       BitOpcode = 63
	BIT_0_B     BitOpcode = 64
	BIT_0_C     BitOpcode = 65
	BIT_0_D     BitOpcode = 66
	BIT_0_E     BitOpcode = 67
	BIT_0_H     BitOpcode = 68
	BIT_0_L     BitOpcode = 69
	BIT_0_PtrHL BitOpcode = 70
	BIT_0_A     BitOpcode = 71
	BIT_1_B     BitOpcode = 72
	BIT_1_C     BitOpcode = 73
	BIT_1_D     BitOpcode = 74
	BIT_1_E     BitOpcode = 75
	BIT_1_H     BitOpcode = 76
	BIT_1_L     BitOpcode = 77
	BIT_1_PtrHL BitOpcode = 78
	BIT_1_A     BitOpcode = 79
	BIT_2_B     BitOpcode = 80
	BIT_2_C     BitOpcode = 81
	BIT_2_D     BitOpcode = 82
	BIT_2_E     BitOpcode = 83
	BIT_2_H     BitOpcode = 84
	BIT_2_L     BitOpcode = 85
	BIT_2_PtrHL BitOpcode = 86
	BIT_2_A     BitOpcode = 87
	BIT_3_B     BitOpcode = 88
	BIT_3_C     BitOpcode = 89
	BIT_3_D     BitOpcode = 90
	BIT_3_E     BitOpcode = 91
	BIT_3_H     BitOpcode = 92
	BIT_3_L     BitOpcode = 93
	BIT_3_PtrHL BitOpcode = 94
	BIT_3_A     BitOpcode = 95
	BIT_4_B     BitOpcode = 96
	BIT_4_C     BitOpcode = 97
	BIT_4_D     BitOpcode = 98
	BIT_4_E     BitOpcode = 99
	BIT_4_H     BitOpcode = 100
	BIT_4_L     BitOpcode = 101
	BIT_4_PtrHL BitOpcode = 102
	BIT_4_A     BitOpcode = 103
	BIT_5_B     BitOpcode = 104
	BIT_5_C     BitOpcode = 105
	BIT_5_D     BitOpcode = 106
	BIT_5_E     BitOpcode = 107
	BIT_5_H     BitOpcode = 108
	BIT_5_L     BitOpcode = 109
	BIT_5_PtrHL BitOpcode = 110
	BIT_5_A     BitOpcode = 111
	BIT_6_B     BitOpcode = 112
	BIT_6_C     BitOpcode = 113
	BIT_6_D     BitOpcode = 114
	BIT_6_E     BitOpcode = 115
	BIT_6_H     BitOpcode = 116
	BIT_6_L     BitOpcode = 117
	BIT_6_PtrHL BitOpcode = 118
	BIT_6_A     BitOpcode = 119
	BIT_7_B     BitOpcode = 120
	BIT_7_C     BitOpcode = 121
	BIT_7_D     BitOpcode = 122
	BIT_7_E     BitOpcode = 123
	BIT_7_H     BitOpcode = 124
	BIT_7_L     BitOpcode = 125
	BIT_7_PtrHL BitOpcode = 126
	BIT_7_A     BitOpcode = 127
	RES_0_B     BitOpcode = 128
	RES_0_C     BitOpcode = 129
	RES_0_D     BitOpcode = 130
	RES_0_E     BitOpcode = 131
	RES_0_H     BitOpcode = 132
	RES_0_L     BitOpcode = 133
	RES_0_PtrHL BitOpcode = 134
	RES_0_A     BitOpcode = 135
	RES_1_B     BitOpcode = 136
	RES_1_C     BitOpcode = 137
	RES_1_D     BitOpcode = 138
	RES_1_E     BitOpcode = 139
	RES_1_H     BitOpcode = 140
	RES_1_L     BitOpcode = 141
	RES_1_PtrHL BitOpcode = 142
	RES_1_A     BitOpcode = 143
	RES_2_B     BitOpcode = 144
	RES_2_C     BitOpcode = 145
	RES_2_D     BitOpcode = 146
	RES_2_E     BitOpcode = 147
	RES_2_H     BitOpcode = 148
	RES_2_L     BitOpcode = 149
	RES_2_PtrHL BitOpcode = 150
	RES_2_A     BitOpcode = 151
	RES_3_B     BitOpcode = 152
	RES_3_C     BitOpcode = 153
	RES_3_D     BitOpcode = 154
	RES_3_E     BitOpcode = 155
	RES_3_H     BitOpcode = 156
	RES_3_L     BitOpcode = 157
	RES_3_PtrHL BitOpcode = 158
	RES_3_A     BitOpcode = 159
	RES_4_B     BitOpcode = 160
	RES_4_C     BitOpcode = 161
	RES_4_D     BitOpcode = 162
	RES_4_E     BitOpcode = 163
	RES_4_H     BitOpcode = 164
	RES_4_L     BitOpcode = 165
	RES_4_PtrHL BitOpcode = 166
	RES_4_A     BitOpcode = 167
	RES_5_B     BitOpcode = 168
	RES_5_C     BitOpcode = 169
	RES_5_D     BitOpcode = 170
	RES_5_E     BitOpcode = 171
	RES_5_H     BitOpcode = 172
	RES_5_L     BitOpcode = 173
	RES_5_PtrHL BitOpcode = 174
	RES_5_A     BitOpcode = 175
	RES_6_B     BitOpcode = 176
	RES_6_C     BitOpcode = 177
	RES_6_D     BitOpcode = 178
	RES_6_E     BitOpcode = 179
	RES_6_H     BitOpcode = 180
	RES_6_L     BitOpcode = 181
	RES_6_PtrHL BitOpcode = 182
	RES_6_A     BitOpcode = 183
	RES_7_B     BitOpcode = 184
	RES_7_C     BitOpcode = 185
	RES_7_D     BitOpcode = 186
	RES_7_E     BitOpcode = 187
	RES_7_H     BitOpcode = 188
	RES_7_L     BitOpcode = 189
	RES_7_PtrHL BitOpcode = 190
	RES_7_A     BitOpcode = 191
	SET_0_B     BitOpcode = 192
	SET_0_C     BitOpcode = 193
	SET_0_D     BitOpcode = 194
	SET_0_E     BitOpcode = 195
	SET_0_H     BitOpcode = 196
	SET_0_L     BitOpcode = 197
	SET_0_PtrHL BitOpcode = 198
	SET_0_A     BitOpcode = 199
	SET_1_B     BitOpcode = 200
	SET_1_C     BitOpcode = 201
	SET_1_D     BitOpcode = 202
	SET_1_E     BitOpcode = 203
	SET_1_H     BitOpcode = 204
	SET_1_L     BitOpcode = 205
	SET_1_PtrHL BitOpcode = 206
	SET_1_A     BitOpcode = 207
	SET_2_B     BitOpcode = 208
	SET_2_C     BitOpcode = 209
	SET_2_D     BitOpcode = 210
	SET_2_E     BitOpcode = 211
	SET_2_H     BitOpcode = 212
	SET_2_L     BitOpcode = 213
	SET_2_PtrHL BitOpcode = 214
	SET_2_A     BitOpcode = 215
	SET_3_B     BitOpcode = 216
	SET_3_C     BitOpcode = 217
	SET_3_D     BitOpcode = 218
	SET_3_E     BitOpcode = 219
	SET_3_H     BitOpcode = 220
	SET_3_L     BitOpcode = 221
	SET_3_PtrHL BitOpcode = 222
	SET_3_A     BitOpcode = 223
	SET_4_B     BitOpcode = 224
	SET_4_C     BitOpcode = 225
	SET_4_D     BitOpcode = 226
	SET_4_E     BitOpcode = 227
	SET_4_H     BitOpcode = 228
	SET_4_L     BitOpcode = 229
	SET_4_PtrHL BitOpcode = 230
	SET_4_A     BitOpcode = 231
	SET_5_B     BitOpcode = 232
	SET_5_C     BitOpcode = 233
	SET_5_D     BitOpcode = 234
	SET_5_E     BitOpcode = 235
	SET_5_H     BitOpcode = 236
	SET_5_L     BitOpcode = 237
	SET_5_PtrHL BitOpcode = 238
	SET_5_A     BitOpcode = 239
	SET_6_B     BitOpcode = 240
	SET_6_C     BitOpcode = 241
	SET_6_D     BitOpcode = 242
	SET_6_E     BitOpcode = 243
	SET_6_H     BitOpcode = 244
	SET_6_L     BitOpcode = 245
	SET_6_PtrHL BitOpcode = 246
	SET_6_A     BitOpcode = 247
	SET_7_B     BitOpcode = 248
	SET_7_C     BitOpcode = 249
	SET_7_D     BitOpcode = 250
	SET_7_E     BitOpcode = 251
	SET_7_H     BitOpcode = 252
	SET_7_L     BitOpcode = 253
	SET_7_PtrHL BitOpcode = 254
	SET_7_A     BitOpcode = 255
)

func (b BitOpcode) String() string {
	switch b {
	case RLC_B:
		return "RLC_B"
	case RLC_C:
		return "RLC_C"
	case RLC_D:
		return "RLC_D"
	case RLC_E:
		return "RLC_E"
	case RLC_H:
		return "RLC_H"
	case RLC_L:
		return "RLC_L"
	case RLC_PtrHL:
		return "RLC_PtrHL"
	case RLC_A:
		return "RLC_A"
	case RRC_B:
		return "RRC_B"
	case RRC_C:
		return "RRC_C"
	case RRC_D:
		return "RRC_D"
	case RRC_E:
		return "RRC_E"
	case RRC_H:
		return "RRC_H"
	case RRC_L:
		return "RRC_L"
	case RRC_PtrHL:
		return "RRC_PtrHL"
	case RRC_A:
		return "RRC_A"
	case RL_B:
		return "RL_B"
	case RL_C:
		return "RL_C"
	case RL_D:
		return "RL_D"
	case RL_E:
		return "RL_E"
	case RL_H:
		return "RL_H"
	case RL_L:
		return "RL_L"
	case RL_PtrHL:
		return "RL_PtrHL"
	case RL_A:
		return "RL_A"
	case RR_B:
		return "RR_B"
	case RR_C:
		return "RR_C"
	case RR_D:
		return "RR_D"
	case RR_E:
		return "RR_E"
	case RR_H:
		return "RR_H"
	case RR_L:
		return "RR_L"
	case RR_PtrHL:
		return "RR_PtrHL"
	case RR_A:
		return "RR_A"
	case SLA_B:
		return "SLA_B"
	case SLA_C:
		return "SLA_C"
	case SLA_D:
		return "SLA_D"
	case SLA_E:
		return "SLA_E"
	case SLA_H:
		return "SLA_H"
	case SLA_L:
		return "SLA_L"
	case SLA_PtrHL:
		return "SLA_PtrHL"
	case SLA_A:
		return "SLA_A"
	case SRA_B:
		return "SRA_B"
	case SRA_C:
		return "SRA_C"
	case SRA_D:
		return "SRA_D"
	case SRA_E:
		return "SRA_E"
	case SRA_H:
		return "SRA_H"
	case SRA_L:
		return "SRA_L"
	case SRA_PtrHL:
		return "SRA_PtrHL"
	case SRA_A:
		return "SRA_A"
	case SLL_B:
		return "SLL_B"
	case SLL_C:
		return "SLL_C"
	case SLL_D:
		return "SLL_D"
	case SLL_E:
		return "SLL_E"
	case SLL_H:
		return "SLL_H"
	case SLL_L:
		return "SLL_L"
	case SLL_PtrHL:
		return "SLL_PtrHL"
	case SLL_A:
		return "SLL_A"
	case SRL_B:
		return "SRL_B"
	case SRL_C:
		return "SRL_C"
	case SRL_D:
		return "SRL_D"
	case SRL_E:
		return "SRL_E"
	case SRL_H:
		return "SRL_H"
	case SRL_L:
		return "SRL_L"
	case SRL_PtrHL:
		return "SRL_PtrHL"
	case SRL_A:
		return "SRL_A"
	case BIT_0_B:
		return "BIT_0_B"
	case BIT_0_C:
		return "BIT_0_C"
	case BIT_0_D:
		return "BIT_0_D"
	case BIT_0_E:
		return "BIT_0_E"
	case BIT_0_H:
		return "BIT_0_H"
	case BIT_0_L:
		return "BIT_0_L"
	case BIT_0_PtrHL:
		return "BIT_0_PtrHL"
	case BIT_0_A:
		return "BIT_0_A"
	case BIT_1_B:
		return "BIT_1_B"
	case BIT_1_C:
		return "BIT_1_C"
	case BIT_1_D:
		return "BIT_1_D"
	case BIT_1_E:
		return "BIT_1_E"
	case BIT_1_H:
		return "BIT_1_H"
	case BIT_1_L:
		return "BIT_1_L"
	case BIT_1_PtrHL:
		return "BIT_1_PtrHL"
	case BIT_1_A:
		return "BIT_1_A"
	case BIT_2_B:
		return "BIT_2_B"
	case BIT_2_C:
		return "BIT_2_C"
	case BIT_2_D:
		return "BIT_2_D"
	case BIT_2_E:
		return "BIT_2_E"
	case BIT_2_H:
		return "BIT_2_H"
	case BIT_2_L:
		return "BIT_2_L"
	case BIT_2_PtrHL:
		return "BIT_2_PtrHL"
	case BIT_2_A:
		return "BIT_2_A"
	case BIT_3_B:
		return "BIT_3_B"
	case BIT_3_C:
		return "BIT_3_C"
	case BIT_3_D:
		return "BIT_3_D"
	case BIT_3_E:
		return "BIT_3_E"
	case BIT_3_H:
		return "BIT_3_H"
	case BIT_3_L:
		return "BIT_3_L"
	case BIT_3_PtrHL:
		return "BIT_3_PtrHL"
	case BIT_3_A:
		return "BIT_3_A"
	case BIT_4_B:
		return "BIT_4_B"
	case BIT_4_C:
		return "BIT_4_C"
	case BIT_4_D:
		return "BIT_4_D"
	case BIT_4_E:
		return "BIT_4_E"
	case BIT_4_H:
		return "BIT_4_H"
	case BIT_4_L:
		return "BIT_4_L"
	case BIT_4_PtrHL:
		return "BIT_4_PtrHL"
	case BIT_4_A:
		return "BIT_4_A"
	case BIT_5_B:
		return "BIT_5_B"
	case BIT_5_C:
		return "BIT_5_C"
	case BIT_5_D:
		return "BIT_5_D"
	case BIT_5_E:
		return "BIT_5_E"
	case BIT_5_H:
		return "BIT_5_H"
	case BIT_5_L:
		return "BIT_5_L"
	case BIT_5_PtrHL:
		return "BIT_5_PtrHL"
	case BIT_5_A:
		return "BIT_5_A"
	case BIT_6_B:
		return "BIT_6_B"
	case BIT_6_C:
		return "BIT_6_C"
	case BIT_6_D:
		return "BIT_6_D"
	case BIT_6_E:
		return "BIT_6_E"
	case BIT_6_H:
		return "BIT_6_H"
	case BIT_6_L:
		return "BIT_6_L"
	case BIT_6_PtrHL:
		return "BIT_6_PtrHL"
	case BIT_6_A:
		return "BIT_6_A"
	case BIT_7_B:
		return "BIT_7_B"
	case BIT_7_C:
		return "BIT_7_C"
	case BIT_7_D:
		return "BIT_7_D"
	case BIT_7_E:
		return "BIT_7_E"
	case BIT_7_H:
		return "BIT_7_H"
	case BIT_7_L:
		return "BIT_7_L"
	case BIT_7_PtrHL:
		return "BIT_7_PtrHL"
	case BIT_7_A:
		return "BIT_7_A"
	case RES_0_B:
		return "RES_0_B"
	case RES_0_C:
		return "RES_0_C"
	case RES_0_D:
		return "RES_0_D"
	case RES_0_E:
		return "RES_0_E"
	case RES_0_H:
		return "RES_0_H"
	case RES_0_L:
		return "RES_0_L"
	case RES_0_PtrHL:
		return "RES_0_PtrHL"
	case RES_0_A:
		return "RES_0_A"
	case RES_1_B:
		return "RES_1_B"
	case RES_1_C:
		return "RES_1_C"
	case RES_1_D:
		return "RES_1_D"
	case RES_1_E:
		return "RES_1_E"
	case RES_1_H:
		return "RES_1_H"
	case RES_1_L:
		return "RES_1_L"
	case RES_1_PtrHL:
		return "RES_1_PtrHL"
	case RES_1_A:
		return "RES_1_A"
	case RES_2_B:
		return "RES_2_B"
	case RES_2_C:
		return "RES_2_C"
	case RES_2_D:
		return "RES_2_D"
	case RES_2_E:
		return "RES_2_E"
	case RES_2_H:
		return "RES_2_H"
	case RES_2_L:
		return "RES_2_L"
	case RES_2_PtrHL:
		return "RES_2_PtrHL"
	case RES_2_A:
		return "RES_2_A"
	case RES_3_B:
		return "RES_3_B"
	case RES_3_C:
		return "RES_3_C"
	case RES_3_D:
		return "RES_3_D"
	case RES_3_E:
		return "RES_3_E"
	case RES_3_H:
		return "RES_3_H"
	case RES_3_L:
		return "RES_3_L"
	case RES_3_PtrHL:
		return "RES_3_PtrHL"
	case RES_3_A:
		return "RES_3_A"
	case RES_4_B:
		return "RES_4_B"
	case RES_4_C:
		return "RES_4_C"
	case RES_4_D:
		return "RES_4_D"
	case RES_4_E:
		return "RES_4_E"
	case RES_4_H:
		return "RES_4_H"
	case RES_4_L:
		return "RES_4_L"
	case RES_4_PtrHL:
		return "RES_4_PtrHL"
	case RES_4_A:
		return "RES_4_A"
	case RES_5_B:
		return "RES_5_B"
	case RES_5_C:
		return "RES_5_C"
	case RES_5_D:
		return "RES_5_D"
	case RES_5_E:
		return "RES_5_E"
	case RES_5_H:
		return "RES_5_H"
	case RES_5_L:
		return "RES_5_L"
	case RES_5_PtrHL:
		return "RES_5_PtrHL"
	case RES_5_A:
		return "RES_5_A"
	case RES_6_B:
		return "RES_6_B"
	case RES_6_C:
		return "RES_6_C"
	case RES_6_D:
		return "RES_6_D"
	case RES_6_E:
		return "RES_6_E"
	case RES_6_H:
		return "RES_6_H"
	case RES_6_L:
		return "RES_6_L"
	case RES_6_PtrHL:
		return "RES_6_PtrHL"
	case RES_6_A:
		return "RES_6_A"
	case RES_7_B:
		return "RES_7_B"
	case RES_7_C:
		return "RES_7_C"
	case RES_7_D:
		return "RES_7_D"
	case RES_7_E:
		return "RES_7_E"
	case RES_7_H:
		return "RES_7_H"
	case RES_7_L:
		return "RES_7_L"
	case RES_7_PtrHL:
		return "RES_7_PtrHL"
	case RES_7_A:
		return "RES_7_A"
	case SET_0_B:
		return "SET_0_B"
	case SET_0_C:
		return "SET_0_C"
	case SET_0_D:
		return "SET_0_D"
	case SET_0_E:
		return "SET_0_E"
	case SET_0_H:
		return "SET_0_H"
	case SET_0_L:
		return "SET_0_L"
	case SET_0_PtrHL:
		return "SET_0_PtrHL"
	case SET_0_A:
		return "SET_0_A"
	case SET_1_B:
		return "SET_1_B"
	case SET_1_C:
		return "SET_1_C"
	case SET_1_D:
		return "SET_1_D"
	case SET_1_E:
		return "SET_1_E"
	case SET_1_H:
		return "SET_1_H"
	case SET_1_L:
		return "SET_1_L"
	case SET_1_PtrHL:
		return "SET_1_PtrHL"
	case SET_1_A:
		return "SET_1_A"
	case SET_2_B:
		return "SET_2_B"
	case SET_2_C:
		return "SET_2_C"
	case SET_2_D:
		return "SET_2_D"
	case SET_2_E:
		return "SET_2_E"
	case SET_2_H:
		return "SET_2_H"
	case SET_2_L:
		return "SET_2_L"
	case SET_2_PtrHL:
		return "SET_2_PtrHL"
	case SET_2_A:
		return "SET_2_A"
	case SET_3_B:
		return "SET_3_B"
	case SET_3_C:
		return "SET_3_C"
	case SET_3_D:
		return "SET_3_D"
	case SET_3_E:
		return "SET_3_E"
	case SET_3_H:
		return "SET_3_H"
	case SET_3_L:
		return "SET_3_L"
	case SET_3_PtrHL:
		return "SET_3_PtrHL"
	case SET_3_A:
		return "SET_3_A"
	case SET_4_B:
		return "SET_4_B"
	case SET_4_C:
		return "SET_4_C"
	case SET_4_D:
		return "SET_4_D"
	case SET_4_E:
		return "SET_4_E"
	case SET_4_H:
		return "SET_4_H"
	case SET_4_L:
		return "SET_4_L"
	case SET_4_PtrHL:
		return "SET_4_PtrHL"
	case SET_4_A:
		return "SET_4_A"
	case SET_5_B:
		return "SET_5_B"
	case SET_5_C:
		return "SET_5_C"
	case SET_5_D:
		return "SET_5_D"
	case SET_5_E:
		return "SET_5_E"
	case SET_5_H:
		return "SET_5_H"
	case SET_5_L:
		return "SET_5_L"
	case SET_5_PtrHL:
		return "SET_5_PtrHL"
	case SET_5_A:
		return "SET_5_A"
	case SET_6_B:
		return "SET_6_B"
	case SET_6_C:
		return "SET_6_C"
	case SET_6_D:
		return "SET_6_D"
	case SET_6_E:
		return "SET_6_E"
	case SET_6_H:
		return "SET_6_H"
	case SET_6_L:
		return "SET_6_L"
	case SET_6_PtrHL:
		return "SET_6_PtrHL"
	case SET_6_A:
		return "SET_6_A"
	case SET_7_B:
		return "SET_7_B"
	case SET_7_C:
		return "SET_7_C"
	case SET_7_D:
		return "SET_7_D"
	case SET_7_E:
		return "SET_7_E"
	case SET_7_H:
		return "SET_7_H"
	case SET_7_L:
		return "SET_7_L"
	case SET_7_PtrHL:
		return "SET_7_PtrHL"
	case SET_7_A:
		return "SET_7_A"
	default:
		panic("unknown bit opcode")
	}
}
