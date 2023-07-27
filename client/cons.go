package client

const FileRelativePath = "download/"

const (
	DataPiece     = iota // 数据片
	ConnectSvr           // 客户端连接服务端
	Download             // 客户端请求下载文件
	DownloadOK           // 客户端通知服务端自己已经下载完某个文件
	ConnectOthers        // 客户端请求服务端通知其他客户端与其建立连接
	ConnectOwn           // 收到其他客户端请求连接自己
	GetFileInfo          // 从服务端获取文件信息
	Message              // 普通消息

	HeaderLen = 5
)
