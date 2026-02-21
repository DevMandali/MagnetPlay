package torrent

import (
    "fmt"

    metainfo "github.com/anacrolix/torrent/metainfo"
    lt "github.com/anacrolix/torrent"
    pb "server/proto"
)

func toResponse(t *lt.Torrent) *pb.TorrentResponse {
    info := t.Info()
    infoHash := t.InfoHash().String()

    if len(info.Files) == 0 {
        return &pb.TorrentResponse{
            TorrentId: infoHash,
            Name:      info.Name,
            Status:    pb.TorrentStatus_SINGLE_FILE,
        }
    }

    return &pb.TorrentResponse{
        TorrentId: infoHash,
        Name:      info.Name,
        Files:     toFileInfoList(infoHash, info),
        Status:    pb.TorrentStatus_MULTI_FILE,
    }
}

func toFileInfoList(infoHash string, info *metainfo.Info) []*pb.FileInfo {
    files := make([]*pb.FileInfo, len(info.Files))
    for i, f := range info.Files {
        files[i] = &pb.FileInfo{
            Id:   fmt.Sprintf("%s:%d", infoHash, i),
            Name: f.DisplayPath(info),
            Size: f.Length,
        }
    }
    return files
}