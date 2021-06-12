package chain

type Opcode uint8

func NewDataOpcode(dLen int) Opcode {
	if dLen == 0 || dLen >= len(dataOpcodes) {
		panic("invalid data opcode")
	}
	return dataOpcodes[dLen-1]
}

func (o Opcode) String() string {
	return opcodeArray[o].name
}

type opcode struct {
	value Opcode
	name  string
}

const (
	OP_0                   Opcode = 0x00 // 0
	OP_FALSE               Opcode = 0x00 // 0 - AKA OP_0
	OP_DATA_1              Opcode = 0x01 // 1
	OP_DATA_2              Opcode = 0x02 // 2
	OP_DATA_3              Opcode = 0x03 // 3
	OP_DATA_4              Opcode = 0x04 // 4
	OP_DATA_5              Opcode = 0x05 // 5
	OP_DATA_6              Opcode = 0x06 // 6
	OP_DATA_7              Opcode = 0x07 // 7
	OP_DATA_8              Opcode = 0x08 // 8
	OP_DATA_9              Opcode = 0x09 // 9
	OP_DATA_10             Opcode = 0x0a // 10
	OP_DATA_11             Opcode = 0x0b // 11
	OP_DATA_12             Opcode = 0x0c // 12
	OP_DATA_13             Opcode = 0x0d // 13
	OP_DATA_14             Opcode = 0x0e // 14
	OP_DATA_15             Opcode = 0x0f // 15
	OP_DATA_16             Opcode = 0x10 // 16
	OP_DATA_17             Opcode = 0x11 // 17
	OP_DATA_18             Opcode = 0x12 // 18
	OP_DATA_19             Opcode = 0x13 // 19
	OP_DATA_20             Opcode = 0x14 // 20
	OP_DATA_21             Opcode = 0x15 // 21
	OP_DATA_22             Opcode = 0x16 // 22
	OP_DATA_23             Opcode = 0x17 // 23
	OP_DATA_24             Opcode = 0x18 // 24
	OP_DATA_25             Opcode = 0x19 // 25
	OP_DATA_26             Opcode = 0x1a // 26
	OP_DATA_27             Opcode = 0x1b // 27
	OP_DATA_28             Opcode = 0x1c // 28
	OP_DATA_29             Opcode = 0x1d // 29
	OP_DATA_30             Opcode = 0x1e // 30
	OP_DATA_31             Opcode = 0x1f // 31
	OP_DATA_32             Opcode = 0x20 // 32
	OP_DATA_33             Opcode = 0x21 // 33
	OP_DATA_34             Opcode = 0x22 // 34
	OP_DATA_35             Opcode = 0x23 // 35
	OP_DATA_36             Opcode = 0x24 // 36
	OP_DATA_37             Opcode = 0x25 // 37
	OP_DATA_38             Opcode = 0x26 // 38
	OP_DATA_39             Opcode = 0x27 // 39
	OP_DATA_40             Opcode = 0x28 // 40
	OP_DATA_41             Opcode = 0x29 // 41
	OP_DATA_42             Opcode = 0x2a // 42
	OP_DATA_43             Opcode = 0x2b // 43
	OP_DATA_44             Opcode = 0x2c // 44
	OP_DATA_45             Opcode = 0x2d // 45
	OP_DATA_46             Opcode = 0x2e // 46
	OP_DATA_47             Opcode = 0x2f // 47
	OP_DATA_48             Opcode = 0x30 // 48
	OP_DATA_49             Opcode = 0x31 // 49
	OP_DATA_50             Opcode = 0x32 // 50
	OP_DATA_51             Opcode = 0x33 // 51
	OP_DATA_52             Opcode = 0x34 // 52
	OP_DATA_53             Opcode = 0x35 // 53
	OP_DATA_54             Opcode = 0x36 // 54
	OP_DATA_55             Opcode = 0x37 // 55
	OP_DATA_56             Opcode = 0x38 // 56
	OP_DATA_57             Opcode = 0x39 // 57
	OP_DATA_58             Opcode = 0x3a // 58
	OP_DATA_59             Opcode = 0x3b // 59
	OP_DATA_60             Opcode = 0x3c // 60
	OP_DATA_61             Opcode = 0x3d // 61
	OP_DATA_62             Opcode = 0x3e // 62
	OP_DATA_63             Opcode = 0x3f // 63
	OP_DATA_64             Opcode = 0x40 // 64
	OP_DATA_65             Opcode = 0x41 // 65
	OP_DATA_66             Opcode = 0x42 // 66
	OP_DATA_67             Opcode = 0x43 // 67
	OP_DATA_68             Opcode = 0x44 // 68
	OP_DATA_69             Opcode = 0x45 // 69
	OP_DATA_70             Opcode = 0x46 // 70
	OP_DATA_71             Opcode = 0x47 // 71
	OP_DATA_72             Opcode = 0x48 // 72
	OP_DATA_73             Opcode = 0x49 // 73
	OP_DATA_74             Opcode = 0x4a // 74
	OP_DATA_75             Opcode = 0x4b // 75
	OP_PUSHDATA1           Opcode = 0x4c // 76
	OP_PUSHDATA2           Opcode = 0x4d // 77
	OP_PUSHDATA4           Opcode = 0x4e // 78
	OP_1NEGATE             Opcode = 0x4f // 79
	OP_RESERVED            Opcode = 0x50 // 80
	OP_1                   Opcode = 0x51 // 81 - AKA OP_TRUE
	OP_TRUE                Opcode = 0x51 // 81
	OP_2                   Opcode = 0x52 // 82
	OP_3                   Opcode = 0x53 // 83
	OP_4                   Opcode = 0x54 // 84
	OP_5                   Opcode = 0x55 // 85
	OP_6                   Opcode = 0x56 // 86
	OP_7                   Opcode = 0x57 // 87
	OP_8                   Opcode = 0x58 // 88
	OP_9                   Opcode = 0x59 // 89
	OP_10                  Opcode = 0x5a // 90
	OP_11                  Opcode = 0x5b // 91
	OP_12                  Opcode = 0x5c // 92
	OP_13                  Opcode = 0x5d // 93
	OP_14                  Opcode = 0x5e // 94
	OP_15                  Opcode = 0x5f // 95
	OP_16                  Opcode = 0x60 // 96
	OP_NOP                 Opcode = 0x61 // 97
	OP_VER                 Opcode = 0x62 // 98
	OP_IF                  Opcode = 0x63 // 99
	OP_NOTIF               Opcode = 0x64 // 100
	OP_VERIF               Opcode = 0x65 // 101
	OP_VERNOTIF            Opcode = 0x66 // 102
	OP_ELSE                Opcode = 0x67 // 103
	OP_ENDIF               Opcode = 0x68 // 104
	OP_VERIFY              Opcode = 0x69 // 105
	OP_RETURN              Opcode = 0x6a // 106
	OP_TOALTSTACK          Opcode = 0x6b // 107
	OP_FROMALTSTACK        Opcode = 0x6c // 108
	OP_2DROP               Opcode = 0x6d // 109
	OP_2DUP                Opcode = 0x6e // 110
	OP_3DUP                Opcode = 0x6f // 111
	OP_2OVER               Opcode = 0x70 // 112
	OP_2ROT                Opcode = 0x71 // 113
	OP_2SWAP               Opcode = 0x72 // 114
	OP_IFDUP               Opcode = 0x73 // 115
	OP_DEPTH               Opcode = 0x74 // 116
	OP_DROP                Opcode = 0x75 // 117
	OP_DUP                 Opcode = 0x76 // 118
	OP_NIP                 Opcode = 0x77 // 119
	OP_OVER                Opcode = 0x78 // 120
	OP_PICK                Opcode = 0x79 // 121
	OP_ROLL                Opcode = 0x7a // 122
	OP_ROT                 Opcode = 0x7b // 123
	OP_SWAP                Opcode = 0x7c // 124
	OP_TUCK                Opcode = 0x7d // 125
	OP_CAT                 Opcode = 0x7e // 126
	OP_SUBSTR              Opcode = 0x7f // 127
	OP_LEFT                Opcode = 0x80 // 128
	OP_RIGHT               Opcode = 0x81 // 129
	OP_SIZE                Opcode = 0x82 // 130
	OP_INVERT              Opcode = 0x83 // 131
	OP_AND                 Opcode = 0x84 // 132
	OP_OR                  Opcode = 0x85 // 133
	OP_XOR                 Opcode = 0x86 // 134
	OP_EQUAL               Opcode = 0x87 // 135
	OP_EQUALVERIFY         Opcode = 0x88 // 136
	OP_RESERVED1           Opcode = 0x89 // 137
	OP_RESERVED2           Opcode = 0x8a // 138
	OP_1ADD                Opcode = 0x8b // 139
	OP_1SUB                Opcode = 0x8c // 140
	OP_2MUL                Opcode = 0x8d // 141
	OP_2DIV                Opcode = 0x8e // 142
	OP_NEGATE              Opcode = 0x8f // 143
	OP_ABS                 Opcode = 0x90 // 144
	OP_NOT                 Opcode = 0x91 // 145
	OP_0NOTEQUAL           Opcode = 0x92 // 146
	OP_ADD                 Opcode = 0x93 // 147
	OP_SUB                 Opcode = 0x94 // 148
	OP_MUL                 Opcode = 0x95 // 149
	OP_DIV                 Opcode = 0x96 // 150
	OP_MOD                 Opcode = 0x97 // 151
	OP_LSHIFT              Opcode = 0x98 // 152
	OP_RSHIFT              Opcode = 0x99 // 153
	OP_BOOLAND             Opcode = 0x9a // 154
	OP_BOOLOR              Opcode = 0x9b // 155
	OP_NUMEQUAL            Opcode = 0x9c // 156
	OP_NUMEQUALVERIFY      Opcode = 0x9d // 157
	OP_NUMNOTEQUAL         Opcode = 0x9e // 158
	OP_LESSTHAN            Opcode = 0x9f // 159
	OP_GREATERTHAN         Opcode = 0xa0 // 160
	OP_LESSTHANOREQUAL     Opcode = 0xa1 // 161
	OP_GREATERTHANOREQUAL  Opcode = 0xa2 // 162
	OP_MIN                 Opcode = 0xa3 // 163
	OP_MAX                 Opcode = 0xa4 // 164
	OP_WITHIN              Opcode = 0xa5 // 165
	OP_RIPEMD160           Opcode = 0xa6 // 166
	OP_SHA1                Opcode = 0xa7 // 167
	OP_SHA256              Opcode = 0xa8 // 168
	OP_HASH160             Opcode = 0xa9 // 169
	OP_HASH256             Opcode = 0xaa // 170
	OP_CODESEPARATOR       Opcode = 0xab // 171
	OP_CHECKSIG            Opcode = 0xac // 172
	OP_CHECKSIGVERIFY      Opcode = 0xad // 173
	OP_CHECKMULTISIG       Opcode = 0xae // 174
	OP_CHECKMULTISIGVERIFY Opcode = 0xaf // 175
	OP_NOP1                Opcode = 0xb0 // 176
	OP_NOP2                Opcode = 0xb1 // 177
	OP_CHECKLOCKTIMEVERIFY Opcode = 0xb1 // 177 - AKA OP_NOP2
	OP_NOP3                Opcode = 0xb2 // 178
	OP_CHECKSEQUENCEVERIFY Opcode = 0xb2 // 178 - AKA OP_NOP3
	OP_NOP4                Opcode = 0xb3 // 179
	OP_NOP5                Opcode = 0xb4 // 180
	OP_NOP6                Opcode = 0xb5 // 181
	OP_NOP7                Opcode = 0xb6 // 182
	OP_NOP8                Opcode = 0xb7 // 183
	OP_NOP9                Opcode = 0xb8 // 184
	OP_NOP10               Opcode = 0xb9 // 185
	OP_UNKNOWN186          Opcode = 0xba // 186
	OP_UNKNOWN187          Opcode = 0xbb // 187
	OP_UNKNOWN188          Opcode = 0xbc // 188
	OP_UNKNOWN189          Opcode = 0xbd // 189
	OP_UNKNOWN190          Opcode = 0xbe // 190
	OP_UNKNOWN191          Opcode = 0xbf // 191
	OP_BLAKE160            Opcode = 0xc0 // 192
	OP_BLAKE256            Opcode = 0xc1 // 193
	OP_SHA3                Opcode = 0xc2 // 194
	OP_KECCAK              Opcode = 0xc3 // 195
	OP_UNKNOWN196          Opcode = 0xc4 // 196
	OP_UNKNOWN197          Opcode = 0xc5 // 197
	OP_UNKNOWN198          Opcode = 0xc6 // 198
	OP_UNKNOWN199          Opcode = 0xc7 // 199
	OP_UNKNOWN200          Opcode = 0xc8 // 200
	OP_UNKNOWN201          Opcode = 0xc9 // 201
	OP_UNKNOWN202          Opcode = 0xca // 202
	OP_UNKNOWN203          Opcode = 0xcb // 203
	OP_UNKNOWN204          Opcode = 0xcc // 204
	OP_UNKNOWN205          Opcode = 0xcd // 205
	OP_UNKNOWN206          Opcode = 0xce // 206
	OP_UNKNOWN207          Opcode = 0xcf // 207
	OP_TYPE                Opcode = 0xd0 // 208
	OP_UNKNOWN209          Opcode = 0xd1 // 209
	OP_UNKNOWN210          Opcode = 0xd2 // 210
	OP_UNKNOWN211          Opcode = 0xd3 // 211
	OP_UNKNOWN212          Opcode = 0xd4 // 212
	OP_UNKNOWN213          Opcode = 0xd5 // 213
	OP_UNKNOWN214          Opcode = 0xd6 // 214
	OP_UNKNOWN215          Opcode = 0xd7 // 215
	OP_UNKNOWN216          Opcode = 0xd8 // 216
	OP_UNKNOWN217          Opcode = 0xd9 // 217
	OP_UNKNOWN218          Opcode = 0xda // 218
	OP_UNKNOWN219          Opcode = 0xdb // 219
	OP_UNKNOWN220          Opcode = 0xdc // 220
	OP_UNKNOWN221          Opcode = 0xdd // 221
	OP_UNKNOWN222          Opcode = 0xde // 222
	OP_UNKNOWN223          Opcode = 0xdf // 223
	OP_UNKNOWN224          Opcode = 0xe0 // 224
	OP_UNKNOWN225          Opcode = 0xe1 // 225
	OP_UNKNOWN226          Opcode = 0xe2 // 226
	OP_UNKNOWN227          Opcode = 0xe3 // 227
	OP_UNKNOWN228          Opcode = 0xe4 // 228
	OP_UNKNOWN229          Opcode = 0xe5 // 229
	OP_UNKNOWN230          Opcode = 0xe6 // 230
	OP_UNKNOWN231          Opcode = 0xe7 // 231
	OP_UNKNOWN232          Opcode = 0xe8 // 232
	OP_UNKNOWN233          Opcode = 0xe9 // 233
	OP_UNKNOWN234          Opcode = 0xea // 234
	OP_UNKNOWN235          Opcode = 0xeb // 235
	OP_UNKNOWN236          Opcode = 0xec // 236
	OP_UNKNOWN237          Opcode = 0xed // 237
	OP_UNKNOWN238          Opcode = 0xee // 238
	OP_UNKNOWN239          Opcode = 0xef // 239
	OP_UNKNOWN240          Opcode = 0xf0 // 240
	OP_UNKNOWN241          Opcode = 0xf1 // 241
	OP_UNKNOWN242          Opcode = 0xf2 // 242
	OP_UNKNOWN243          Opcode = 0xf3 // 243
	OP_UNKNOWN244          Opcode = 0xf4 // 244
	OP_UNKNOWN245          Opcode = 0xf5 // 245
	OP_UNKNOWN246          Opcode = 0xf6 // 246
	OP_UNKNOWN247          Opcode = 0xf7 // 247
	OP_UNKNOWN248          Opcode = 0xf8 // 248
	OP_UNKNOWN249          Opcode = 0xf9 // 249
	OP_SMALLINTEGER        Opcode = 0xfa // 250 - bitcoin core internal
	OP_PUBKEYS             Opcode = 0xfb // 251 - bitcoin core internal
	OP_UNKNOWN252          Opcode = 0xfc // 252
	OP_PUBKEYHASH          Opcode = 0xfd // 253 - bitcoin core internal
	OP_PUBKEY              Opcode = 0xfe // 254 - bitcoin core internal
	OP_INVALIDOPCODE       Opcode = 0xff // 255 - bitcoin core internal
)

var opcodeArray = [256]opcode{
	// Data push opcodes.
	OP_FALSE:     {OP_FALSE, "OP_0"},
	OP_DATA_1:    {OP_DATA_1, "OP_DATA_1"},
	OP_DATA_2:    {OP_DATA_2, "OP_DATA_2"},
	OP_DATA_3:    {OP_DATA_3, "OP_DATA_3"},
	OP_DATA_4:    {OP_DATA_4, "OP_DATA_4"},
	OP_DATA_5:    {OP_DATA_5, "OP_DATA_5"},
	OP_DATA_6:    {OP_DATA_6, "OP_DATA_6"},
	OP_DATA_7:    {OP_DATA_7, "OP_DATA_7"},
	OP_DATA_8:    {OP_DATA_8, "OP_DATA_8"},
	OP_DATA_9:    {OP_DATA_9, "OP_DATA_9"},
	OP_DATA_10:   {OP_DATA_10, "OP_DATA_10"},
	OP_DATA_11:   {OP_DATA_11, "OP_DATA_11"},
	OP_DATA_12:   {OP_DATA_12, "OP_DATA_12"},
	OP_DATA_13:   {OP_DATA_13, "OP_DATA_13"},
	OP_DATA_14:   {OP_DATA_14, "OP_DATA_14"},
	OP_DATA_15:   {OP_DATA_15, "OP_DATA_15"},
	OP_DATA_16:   {OP_DATA_16, "OP_DATA_16"},
	OP_DATA_17:   {OP_DATA_17, "OP_DATA_17"},
	OP_DATA_18:   {OP_DATA_18, "OP_DATA_18"},
	OP_DATA_19:   {OP_DATA_19, "OP_DATA_19"},
	OP_DATA_20:   {OP_DATA_20, "OP_DATA_20"},
	OP_DATA_21:   {OP_DATA_21, "OP_DATA_21"},
	OP_DATA_22:   {OP_DATA_22, "OP_DATA_22"},
	OP_DATA_23:   {OP_DATA_23, "OP_DATA_23"},
	OP_DATA_24:   {OP_DATA_24, "OP_DATA_24"},
	OP_DATA_25:   {OP_DATA_25, "OP_DATA_25"},
	OP_DATA_26:   {OP_DATA_26, "OP_DATA_26"},
	OP_DATA_27:   {OP_DATA_27, "OP_DATA_27"},
	OP_DATA_28:   {OP_DATA_28, "OP_DATA_28"},
	OP_DATA_29:   {OP_DATA_29, "OP_DATA_29"},
	OP_DATA_30:   {OP_DATA_30, "OP_DATA_30"},
	OP_DATA_31:   {OP_DATA_31, "OP_DATA_31"},
	OP_DATA_32:   {OP_DATA_32, "OP_DATA_32"},
	OP_DATA_33:   {OP_DATA_33, "OP_DATA_33"},
	OP_DATA_34:   {OP_DATA_34, "OP_DATA_34"},
	OP_DATA_35:   {OP_DATA_35, "OP_DATA_35"},
	OP_DATA_36:   {OP_DATA_36, "OP_DATA_36"},
	OP_DATA_37:   {OP_DATA_37, "OP_DATA_37"},
	OP_DATA_38:   {OP_DATA_38, "OP_DATA_38"},
	OP_DATA_39:   {OP_DATA_39, "OP_DATA_39"},
	OP_DATA_40:   {OP_DATA_40, "OP_DATA_40"},
	OP_DATA_41:   {OP_DATA_41, "OP_DATA_41"},
	OP_DATA_42:   {OP_DATA_42, "OP_DATA_42"},
	OP_DATA_43:   {OP_DATA_43, "OP_DATA_43"},
	OP_DATA_44:   {OP_DATA_44, "OP_DATA_44"},
	OP_DATA_45:   {OP_DATA_45, "OP_DATA_45"},
	OP_DATA_46:   {OP_DATA_46, "OP_DATA_46"},
	OP_DATA_47:   {OP_DATA_47, "OP_DATA_47"},
	OP_DATA_48:   {OP_DATA_48, "OP_DATA_48"},
	OP_DATA_49:   {OP_DATA_49, "OP_DATA_49"},
	OP_DATA_50:   {OP_DATA_50, "OP_DATA_50"},
	OP_DATA_51:   {OP_DATA_51, "OP_DATA_51"},
	OP_DATA_52:   {OP_DATA_52, "OP_DATA_52"},
	OP_DATA_53:   {OP_DATA_53, "OP_DATA_53"},
	OP_DATA_54:   {OP_DATA_54, "OP_DATA_54"},
	OP_DATA_55:   {OP_DATA_55, "OP_DATA_55"},
	OP_DATA_56:   {OP_DATA_56, "OP_DATA_56"},
	OP_DATA_57:   {OP_DATA_57, "OP_DATA_57"},
	OP_DATA_58:   {OP_DATA_58, "OP_DATA_58"},
	OP_DATA_59:   {OP_DATA_59, "OP_DATA_59"},
	OP_DATA_60:   {OP_DATA_60, "OP_DATA_60"},
	OP_DATA_61:   {OP_DATA_61, "OP_DATA_61"},
	OP_DATA_62:   {OP_DATA_62, "OP_DATA_62"},
	OP_DATA_63:   {OP_DATA_63, "OP_DATA_63"},
	OP_DATA_64:   {OP_DATA_64, "OP_DATA_64"},
	OP_DATA_65:   {OP_DATA_65, "OP_DATA_65"},
	OP_DATA_66:   {OP_DATA_66, "OP_DATA_66"},
	OP_DATA_67:   {OP_DATA_67, "OP_DATA_67"},
	OP_DATA_68:   {OP_DATA_68, "OP_DATA_68"},
	OP_DATA_69:   {OP_DATA_69, "OP_DATA_69"},
	OP_DATA_70:   {OP_DATA_70, "OP_DATA_70"},
	OP_DATA_71:   {OP_DATA_71, "OP_DATA_71"},
	OP_DATA_72:   {OP_DATA_72, "OP_DATA_72"},
	OP_DATA_73:   {OP_DATA_73, "OP_DATA_73"},
	OP_DATA_74:   {OP_DATA_74, "OP_DATA_74"},
	OP_DATA_75:   {OP_DATA_75, "OP_DATA_75"},
	OP_PUSHDATA1: {OP_PUSHDATA1, "OP_PUSHDATA1"},
	OP_PUSHDATA2: {OP_PUSHDATA2, "OP_PUSHDATA2"},
	OP_PUSHDATA4: {OP_PUSHDATA4, "OP_PUSHDATA4"},
	OP_1NEGATE:   {OP_1NEGATE, "OP_1NEGATE"},
	OP_RESERVED:  {OP_RESERVED, "OP_RESERVED"},
	OP_TRUE:      {OP_TRUE, "OP_1"},
	OP_2:         {OP_2, "OP_2"},
	OP_3:         {OP_3, "OP_3"},
	OP_4:         {OP_4, "OP_4"},
	OP_5:         {OP_5, "OP_5"},
	OP_6:         {OP_6, "OP_6"},
	OP_7:         {OP_7, "OP_7"},
	OP_8:         {OP_8, "OP_8"},
	OP_9:         {OP_9, "OP_9"},
	OP_10:        {OP_10, "OP_10"},
	OP_11:        {OP_11, "OP_11"},
	OP_12:        {OP_12, "OP_12"},
	OP_13:        {OP_13, "OP_13"},
	OP_14:        {OP_14, "OP_14"},
	OP_15:        {OP_15, "OP_15"},
	OP_16:        {OP_16, "OP_16"},

	// Control opcodes.
	OP_NOP:                 {OP_NOP, "OP_NOP"},
	OP_VER:                 {OP_VER, "OP_VER"},
	OP_IF:                  {OP_IF, "OP_IF"},
	OP_NOTIF:               {OP_NOTIF, "OP_NOTIF"},
	OP_VERIF:               {OP_VERIF, "OP_VERIF"},
	OP_VERNOTIF:            {OP_VERNOTIF, "OP_VERNOTIF"},
	OP_ELSE:                {OP_ELSE, "OP_ELSE"},
	OP_ENDIF:               {OP_ENDIF, "OP_ENDIF"},
	OP_VERIFY:              {OP_VERIFY, "OP_VERIFY"},
	OP_RETURN:              {OP_RETURN, "OP_RETURN"},
	OP_CHECKLOCKTIMEVERIFY: {OP_CHECKLOCKTIMEVERIFY, "OP_CHECKLOCKTIMEVERIFY"},
	OP_CHECKSEQUENCEVERIFY: {OP_CHECKSEQUENCEVERIFY, "OP_CHECKSEQUENCEVERIFY"},

	// Stack opcodes.
	OP_TOALTSTACK:   {OP_TOALTSTACK, "OP_TOALTSTACK"},
	OP_FROMALTSTACK: {OP_FROMALTSTACK, "OP_FROMALTSTACK"},
	OP_2DROP:        {OP_2DROP, "OP_2DROP"},
	OP_2DUP:         {OP_2DUP, "OP_2DUP"},
	OP_3DUP:         {OP_3DUP, "OP_3DUP"},
	OP_2OVER:        {OP_2OVER, "OP_2OVER"},
	OP_2ROT:         {OP_2ROT, "OP_2ROT"},
	OP_2SWAP:        {OP_2SWAP, "OP_2SWAP"},
	OP_IFDUP:        {OP_IFDUP, "OP_IFDUP"},
	OP_DEPTH:        {OP_DEPTH, "OP_DEPTH"},
	OP_DROP:         {OP_DROP, "OP_DROP"},
	OP_DUP:          {OP_DUP, "OP_DUP"},
	OP_NIP:          {OP_NIP, "OP_NIP"},
	OP_OVER:         {OP_OVER, "OP_OVER"},
	OP_PICK:         {OP_PICK, "OP_PICK"},
	OP_ROLL:         {OP_ROLL, "OP_ROLL"},
	OP_ROT:          {OP_ROT, "OP_ROT"},
	OP_SWAP:         {OP_SWAP, "OP_SWAP"},
	OP_TUCK:         {OP_TUCK, "OP_TUCK"},

	// Splice opcodes.
	OP_CAT:    {OP_CAT, "OP_CAT"},
	OP_SUBSTR: {OP_SUBSTR, "OP_SUBSTR"},
	OP_LEFT:   {OP_LEFT, "OP_LEFT"},
	OP_RIGHT:  {OP_RIGHT, "OP_RIGHT"},
	OP_SIZE:   {OP_SIZE, "OP_SIZE"},

	// Bitwise logic opcodes.
	OP_INVERT:      {OP_INVERT, "OP_INVERT"},
	OP_AND:         {OP_AND, "OP_AND"},
	OP_OR:          {OP_OR, "OP_OR"},
	OP_XOR:         {OP_XOR, "OP_XOR"},
	OP_EQUAL:       {OP_EQUAL, "OP_EQUAL"},
	OP_EQUALVERIFY: {OP_EQUALVERIFY, "OP_EQUALVERIFY"},
	OP_RESERVED1:   {OP_RESERVED1, "OP_RESERVED1"},
	OP_RESERVED2:   {OP_RESERVED2, "OP_RESERVED2"},

	// Numeric related opcodes.
	OP_1ADD:               {OP_1ADD, "OP_1ADD"},
	OP_1SUB:               {OP_1SUB, "OP_1SUB"},
	OP_2MUL:               {OP_2MUL, "OP_2MUL"},
	OP_2DIV:               {OP_2DIV, "OP_2DIV"},
	OP_NEGATE:             {OP_NEGATE, "OP_NEGATE"},
	OP_ABS:                {OP_ABS, "OP_ABS"},
	OP_NOT:                {OP_NOT, "OP_NOT"},
	OP_0NOTEQUAL:          {OP_0NOTEQUAL, "OP_0NOTEQUAL"},
	OP_ADD:                {OP_ADD, "OP_ADD"},
	OP_SUB:                {OP_SUB, "OP_SUB"},
	OP_MUL:                {OP_MUL, "OP_MUL"},
	OP_DIV:                {OP_DIV, "OP_DIV"},
	OP_MOD:                {OP_MOD, "OP_MOD"},
	OP_LSHIFT:             {OP_LSHIFT, "OP_LSHIFT"},
	OP_RSHIFT:             {OP_RSHIFT, "OP_RSHIFT"},
	OP_BOOLAND:            {OP_BOOLAND, "OP_BOOLAND"},
	OP_BOOLOR:             {OP_BOOLOR, "OP_BOOLOR"},
	OP_NUMEQUAL:           {OP_NUMEQUAL, "OP_NUMEQUAL"},
	OP_NUMEQUALVERIFY:     {OP_NUMEQUALVERIFY, "OP_NUMEQUALVERIFY"},
	OP_NUMNOTEQUAL:        {OP_NUMNOTEQUAL, "OP_NUMNOTEQUAL"},
	OP_LESSTHAN:           {OP_LESSTHAN, "OP_LESSTHAN"},
	OP_GREATERTHAN:        {OP_GREATERTHAN, "OP_GREATERTHAN"},
	OP_LESSTHANOREQUAL:    {OP_LESSTHANOREQUAL, "OP_LESSTHANOREQUAL"},
	OP_GREATERTHANOREQUAL: {OP_GREATERTHANOREQUAL, "OP_GREATERTHANOREQUAL"},
	OP_MIN:                {OP_MIN, "OP_MIN"},
	OP_MAX:                {OP_MAX, "OP_MAX"},
	OP_WITHIN:             {OP_WITHIN, "OP_WITHIN"},

	// Crypto opcodes.
	OP_RIPEMD160:           {OP_RIPEMD160, "OP_RIPEMD160"},
	OP_SHA1:                {OP_SHA1, "OP_SHA1"},
	OP_SHA256:              {OP_SHA256, "OP_SHA256"},
	OP_HASH160:             {OP_HASH160, "OP_HASH160"},
	OP_HASH256:             {OP_HASH256, "OP_HASH256"},
	OP_CODESEPARATOR:       {OP_CODESEPARATOR, "OP_CODESEPARATOR"},
	OP_CHECKSIG:            {OP_CHECKSIG, "OP_CHECKSIG"},
	OP_CHECKSIGVERIFY:      {OP_CHECKSIGVERIFY, "OP_CHECKSIGVERIFY"},
	OP_CHECKMULTISIG:       {OP_CHECKMULTISIG, "OP_CHECKMULTISIG"},
	OP_CHECKMULTISIGVERIFY: {OP_CHECKMULTISIGVERIFY, "OP_CHECKMULTISIGVERIFY"},

	// Reserved opcodes.
	OP_NOP1:  {OP_NOP1, "OP_NOP1"},
	OP_NOP4:  {OP_NOP4, "OP_NOP4"},
	OP_NOP5:  {OP_NOP5, "OP_NOP5"},
	OP_NOP6:  {OP_NOP6, "OP_NOP6"},
	OP_NOP7:  {OP_NOP7, "OP_NOP7"},
	OP_NOP8:  {OP_NOP8, "OP_NOP8"},
	OP_NOP9:  {OP_NOP9, "OP_NOP9"},
	OP_NOP10: {OP_NOP10, "OP_NOP10"},

	// Undefined opcodes.
	OP_UNKNOWN186: {OP_UNKNOWN186, "OP_UNKNOWN186"},
	OP_UNKNOWN187: {OP_UNKNOWN187, "OP_UNKNOWN187"},
	OP_UNKNOWN188: {OP_UNKNOWN188, "OP_UNKNOWN188"},
	OP_UNKNOWN189: {OP_UNKNOWN189, "OP_UNKNOWN189"},
	OP_UNKNOWN190: {OP_UNKNOWN190, "OP_UNKNOWN190"},
	OP_UNKNOWN191: {OP_UNKNOWN191, "OP_UNKNOWN191"},
	OP_BLAKE160:   {OP_BLAKE160, "OP_BLAKE160"},
	OP_BLAKE256:   {OP_BLAKE256, "OP_BLAKE256"},
	OP_SHA3:       {OP_SHA3, "OP_SHA3"},
	OP_KECCAK:     {OP_KECCAK, "OP_KECCAK"},
	OP_UNKNOWN196: {OP_UNKNOWN196, "OP_UNKNOWN196"},
	OP_UNKNOWN197: {OP_UNKNOWN197, "OP_UNKNOWN197"},
	OP_UNKNOWN198: {OP_UNKNOWN198, "OP_UNKNOWN198"},
	OP_UNKNOWN199: {OP_UNKNOWN199, "OP_UNKNOWN199"},
	OP_UNKNOWN200: {OP_UNKNOWN200, "OP_UNKNOWN200"},
	OP_UNKNOWN201: {OP_UNKNOWN201, "OP_UNKNOWN201"},
	OP_UNKNOWN202: {OP_UNKNOWN202, "OP_UNKNOWN202"},
	OP_UNKNOWN203: {OP_UNKNOWN203, "OP_UNKNOWN203"},
	OP_UNKNOWN204: {OP_UNKNOWN204, "OP_UNKNOWN204"},
	OP_UNKNOWN205: {OP_UNKNOWN205, "OP_UNKNOWN205"},
	OP_UNKNOWN206: {OP_UNKNOWN206, "OP_UNKNOWN206"},
	OP_UNKNOWN207: {OP_UNKNOWN207, "OP_UNKNOWN207"},
	OP_TYPE:       {OP_TYPE, "OP_UNKNOWN208"},
	OP_UNKNOWN209: {OP_UNKNOWN209, "OP_UNKNOWN209"},
	OP_UNKNOWN210: {OP_UNKNOWN210, "OP_UNKNOWN210"},
	OP_UNKNOWN211: {OP_UNKNOWN211, "OP_UNKNOWN211"},
	OP_UNKNOWN212: {OP_UNKNOWN212, "OP_UNKNOWN212"},
	OP_UNKNOWN213: {OP_UNKNOWN213, "OP_UNKNOWN213"},
	OP_UNKNOWN214: {OP_UNKNOWN214, "OP_UNKNOWN214"},
	OP_UNKNOWN215: {OP_UNKNOWN215, "OP_UNKNOWN215"},
	OP_UNKNOWN216: {OP_UNKNOWN216, "OP_UNKNOWN216"},
	OP_UNKNOWN217: {OP_UNKNOWN217, "OP_UNKNOWN217"},
	OP_UNKNOWN218: {OP_UNKNOWN218, "OP_UNKNOWN218"},
	OP_UNKNOWN219: {OP_UNKNOWN219, "OP_UNKNOWN219"},
	OP_UNKNOWN220: {OP_UNKNOWN220, "OP_UNKNOWN220"},
	OP_UNKNOWN221: {OP_UNKNOWN221, "OP_UNKNOWN221"},
	OP_UNKNOWN222: {OP_UNKNOWN222, "OP_UNKNOWN222"},
	OP_UNKNOWN223: {OP_UNKNOWN223, "OP_UNKNOWN223"},
	OP_UNKNOWN224: {OP_UNKNOWN224, "OP_UNKNOWN224"},
	OP_UNKNOWN225: {OP_UNKNOWN225, "OP_UNKNOWN225"},
	OP_UNKNOWN226: {OP_UNKNOWN226, "OP_UNKNOWN226"},
	OP_UNKNOWN227: {OP_UNKNOWN227, "OP_UNKNOWN227"},
	OP_UNKNOWN228: {OP_UNKNOWN228, "OP_UNKNOWN228"},
	OP_UNKNOWN229: {OP_UNKNOWN229, "OP_UNKNOWN229"},
	OP_UNKNOWN230: {OP_UNKNOWN230, "OP_UNKNOWN230"},
	OP_UNKNOWN231: {OP_UNKNOWN231, "OP_UNKNOWN231"},
	OP_UNKNOWN232: {OP_UNKNOWN232, "OP_UNKNOWN232"},
	OP_UNKNOWN233: {OP_UNKNOWN233, "OP_UNKNOWN233"},
	OP_UNKNOWN234: {OP_UNKNOWN234, "OP_UNKNOWN234"},
	OP_UNKNOWN235: {OP_UNKNOWN235, "OP_UNKNOWN235"},
	OP_UNKNOWN236: {OP_UNKNOWN236, "OP_UNKNOWN236"},
	OP_UNKNOWN237: {OP_UNKNOWN237, "OP_UNKNOWN237"},
	OP_UNKNOWN238: {OP_UNKNOWN238, "OP_UNKNOWN238"},
	OP_UNKNOWN239: {OP_UNKNOWN239, "OP_UNKNOWN239"},
	OP_UNKNOWN240: {OP_UNKNOWN240, "OP_UNKNOWN240"},
	OP_UNKNOWN241: {OP_UNKNOWN241, "OP_UNKNOWN241"},
	OP_UNKNOWN242: {OP_UNKNOWN242, "OP_UNKNOWN242"},
	OP_UNKNOWN243: {OP_UNKNOWN243, "OP_UNKNOWN243"},
	OP_UNKNOWN244: {OP_UNKNOWN244, "OP_UNKNOWN244"},
	OP_UNKNOWN245: {OP_UNKNOWN245, "OP_UNKNOWN245"},
	OP_UNKNOWN246: {OP_UNKNOWN246, "OP_UNKNOWN246"},
	OP_UNKNOWN247: {OP_UNKNOWN247, "OP_UNKNOWN247"},
	OP_UNKNOWN248: {OP_UNKNOWN248, "OP_UNKNOWN248"},
	OP_UNKNOWN249: {OP_UNKNOWN249, "OP_UNKNOWN249"},

	// Bitcoin Core internal use opcode.  Defined here for completeness.
	OP_SMALLINTEGER: {OP_SMALLINTEGER, "OP_SMALLINTEGER"},
	OP_PUBKEYS:      {OP_PUBKEYS, "OP_PUBKEYS"},
	OP_UNKNOWN252:   {OP_UNKNOWN252, "OP_UNKNOWN252"},
	OP_PUBKEYHASH:   {OP_PUBKEYHASH, "OP_PUBKEYHASH"},
	OP_PUBKEY:       {OP_PUBKEY, "OP_PUBKEY"},

	OP_INVALIDOPCODE: {OP_INVALIDOPCODE, "OP_INVALIDOPCODE"},
}

var dataOpcodes = [75]Opcode{
	OP_DATA_1,
	OP_DATA_2,
	OP_DATA_3,
	OP_DATA_4,
	OP_DATA_5,
	OP_DATA_6,
	OP_DATA_7,
	OP_DATA_8,
	OP_DATA_9,
	OP_DATA_10,
	OP_DATA_11,
	OP_DATA_12,
	OP_DATA_13,
	OP_DATA_14,
	OP_DATA_15,
	OP_DATA_16,
	OP_DATA_17,
	OP_DATA_18,
	OP_DATA_19,
	OP_DATA_20,
	OP_DATA_21,
	OP_DATA_22,
	OP_DATA_23,
	OP_DATA_24,
	OP_DATA_25,
	OP_DATA_26,
	OP_DATA_27,
	OP_DATA_28,
	OP_DATA_29,
	OP_DATA_30,
	OP_DATA_31,
	OP_DATA_32,
	OP_DATA_33,
	OP_DATA_34,
	OP_DATA_35,
	OP_DATA_36,
	OP_DATA_37,
	OP_DATA_38,
	OP_DATA_39,
	OP_DATA_40,
	OP_DATA_41,
	OP_DATA_42,
	OP_DATA_43,
	OP_DATA_44,
	OP_DATA_45,
	OP_DATA_46,
	OP_DATA_47,
	OP_DATA_48,
	OP_DATA_49,
	OP_DATA_50,
	OP_DATA_51,
	OP_DATA_52,
	OP_DATA_53,
	OP_DATA_54,
	OP_DATA_55,
	OP_DATA_56,
	OP_DATA_57,
	OP_DATA_58,
	OP_DATA_59,
	OP_DATA_60,
	OP_DATA_61,
	OP_DATA_62,
	OP_DATA_63,
	OP_DATA_64,
	OP_DATA_65,
	OP_DATA_66,
	OP_DATA_67,
	OP_DATA_68,
	OP_DATA_69,
	OP_DATA_70,
	OP_DATA_71,
	OP_DATA_72,
	OP_DATA_73,
	OP_DATA_74,
	OP_DATA_75,
}
