package helper

type String4 [4]byte

func NewString4(v string) (s String4) { copy(s[:], v); return }
func (s String4) String() string      { return string(s[:]) }

type String8 [8]byte

func NewString8(v string) (s String8) { copy(s[:], v); return }
func (s String8) String() string      { return string(s[:]) }

type String16 [16]byte

func NewString16(v string) (s String16) { copy(s[:], v); return }
func (s String16) String() string       { return string(s[:]) }

type String128 [128]byte

func NewString128(v string) (s String128) { copy(s[:], v); return }
func (s String128) String() string        { return string(s[:]) }

type String256 [256]byte

func NewString256(v string) (s String256) { copy(s[:], v); return }
func (s String256) String() string        { return string(s[:]) }
