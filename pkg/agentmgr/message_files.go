package agentmgr

// FileListData is sent from hub to agent to list a directory.
type FileListData struct {
	RequestID  string `json:"request_id"`
	Path       string `json:"path"`
	ShowHidden bool   `json:"show_hidden,omitempty"`
}

// FileEntry describes a single file/directory in a listing or search result.
type FileEntry struct {
	Name    string `json:"name"`
	Path    string `json:"path,omitempty"` // absolute path; populated in search results
	Size    int64  `json:"size"`
	Mode    string `json:"mode"`
	ModTime string `json:"mod_time"`
	IsDir   bool   `json:"is_dir"`
}

// FileListedData is sent from agent to hub with directory listing results.
type FileListedData struct {
	RequestID string      `json:"request_id"`
	Path      string      `json:"path"`
	Entries   []FileEntry `json:"entries"`
	Error     string      `json:"error,omitempty"`
}

// FileReadData is sent from hub to agent to read a file.
type FileReadData struct {
	RequestID string `json:"request_id"`
	Path      string `json:"path"`
}

// FileDataPayload carries file content chunks (base64-encoded).
type FileDataPayload struct {
	RequestID string `json:"request_id"`
	Data      string `json:"data"`   // base64-encoded chunk
	Offset    int64  `json:"offset"` // byte offset
	Done      bool   `json:"done"`   // true if this is the last chunk
	Error     string `json:"error,omitempty"`
}

// FileWriteData is sent from hub to agent to write a file chunk.
type FileWriteData struct {
	RequestID string `json:"request_id"`
	Path      string `json:"path"`
	Data      string `json:"data"`   // base64-encoded chunk
	Offset    int64  `json:"offset"` // byte offset
	Done      bool   `json:"done"`   // true if this is the last chunk
}

// FileWrittenData is sent from agent to hub confirming write.
type FileWrittenData struct {
	RequestID    string `json:"request_id"`
	BytesWritten int64  `json:"bytes_written"`
	Error        string `json:"error,omitempty"`
}

// FileMkdirData is sent from hub to agent to create a directory.
type FileMkdirData struct {
	RequestID string `json:"request_id"`
	Path      string `json:"path"`
}

// FileDeleteData is sent from hub to agent to delete a file or directory.
type FileDeleteData struct {
	RequestID string `json:"request_id"`
	Path      string `json:"path"`
}

// FileRenameData is sent from hub to agent to rename/move a file or directory.
type FileRenameData struct {
	RequestID string `json:"request_id"`
	OldPath   string `json:"old_path"`
	NewPath   string `json:"new_path"`
}

// FileCopyData is sent from hub to agent to copy a file or directory.
type FileCopyData struct {
	RequestID string `json:"request_id"`
	SrcPath   string `json:"src_path"`
	DstPath   string `json:"dst_path"`
}

// FileResultData is a generic result for file operations (mkdir, delete, rename, copy).
type FileResultData struct {
	RequestID string `json:"request_id"`
	OK        bool   `json:"ok"`
	Error     string `json:"error,omitempty"`
}

// FileSearchData is sent from hub to agent to search for files matching a pattern.
type FileSearchData struct {
	RequestID  string `json:"request_id"`
	Path       string `json:"path"`
	Pattern    string `json:"pattern"`     // glob pattern matched against filename (e.g. "*.log")
	MaxResults int    `json:"max_results"` // default 100, capped at 500
}

// FileSearchResultData is sent from agent to hub with the filename search results.
type FileSearchResultData struct {
	RequestID string      `json:"request_id"`
	Matches   []FileEntry `json:"matches"`
	Error     string      `json:"error,omitempty"`
	Truncated bool        `json:"truncated"` // true when results were capped at MaxResults
}
