-- name: CreateDAP :exec
INSERT INTO daps (id, did, handle, proof) VALUES ($1, $2, $3, $4);

-- name: GetDAP :one
SELECT * FROM daps WHERE handle = $1 LIMIT 1;