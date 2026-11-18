package main

import (
	"context"
	"encoding/json"
	"github.com/jackc/pgx/v5"
)

func (s *Server) copyLatest(ctx context.Context, t pgx.Tx, src, dest string) error {
	var oldID string
	var size int64
	e := t.QueryRow(ctx, "SELECT id,size FROM versions WHERE path=$1 ORDER BY created_at DESC,id DESC LIMIT 1", src).Scan(&oldID, &size)
	if e != nil {
		return e
	}
	manifest := Manifest{Path: dest, Size: size, Chunks: []Chunk{}}
	rows, e := t.Query(ctx, "SELECT b.hash,b.size FROM version_blocks v JOIN blocks b ON b.hash=v.hash WHERE v.version_id=$1 ORDER BY v.ordinal", oldID)
	if e != nil {
		return e
	}
	for rows.Next() {
		var c Chunk
		if e = rows.Scan(&c.Hash, &c.Size); e != nil {
			rows.Close()
			return e
		}
		manifest.Chunks = append(manifest.Chunks, c)
	}
	e = rows.Err()
	rows.Close()
	if e != nil {
		return e
	}
	raw, _ := json.Marshal(manifest)
	id := randomID()
	if _, e = t.Exec(ctx, "INSERT INTO uploads(id,idem,request_digest,path,manifest,size,expires_at,committed) VALUES($1,$2,$3,$4,$5,$6,now(),true)", id, "copy:"+id, digest(raw), dest, raw, size); e != nil {
		return e
	}
	if _, e = t.Exec(ctx, "INSERT INTO versions(id,path,size) VALUES($1,$2,$3)", id, dest, size); e != nil {
		return e
	}
	if _, e = t.Exec(ctx, "INSERT INTO version_blocks(version_id,ordinal,hash) SELECT $1,ordinal,hash FROM version_blocks WHERE version_id=$2", id, oldID); e != nil {
		return e
	}
	_, e = t.Exec(ctx, `UPDATE blocks b SET refs=refs+x.n,gc_after=NULL FROM
 (SELECT hash,count(*) n FROM version_blocks WHERE version_id=$1 GROUP BY hash) x WHERE b.hash=x.hash`, id)
	return e
}
