package conf

var (
	//LenStackBuf LenStackBuf
	LenStackBuf = 4096

	//LogLevel log
	LogLevel string
	//LogPath LogPath
	LogPath string
	//LogFlag LogFlag
	LogFlag int

	//ConsolePort console
	ConsolePort int
	//ConsolePrompt ConsolePrompt
	ConsolePrompt = "Leaf# "
	//ProfilePath ProfilePath
	ProfilePath string

	//ListenAddr cluster
	ListenAddr string
	//ConnAddrs ConnAddrs
	ConnAddrs []string
	//PendingWriteNum PendingWriteNum
	PendingWriteNum int
)
