package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"mikan/internal/panel/externalnodes"
	"mikan/internal/panel/settings"
)

type externalNodeView struct {
	ID       int64  `json:"id"`
	Name     string `json:"name"`
	Protocol string `json:"protocol"`
	Enabled  bool   `json:"enabled"`
	Auto     bool   `json:"auto"`
	Server   string `json:"server"`
	Port     int    `json:"port"`
}
type externalNodesOutput struct{ Body []externalNodeView }
type importExternalInput struct {
	Body struct {
		JSON string `json:"json" maxLength:"2097152"`
		Name string `json:"name,omitempty" maxLength:"120"`
	}
}
type externalIDInput struct {
	ID int64 `path:"id" minimum:"1"`
}
type patchExternalInput struct {
	ID   int64 `path:"id" minimum:"1"`
	Body struct {
		Name    *string `json:"name,omitempty" maxLength:"120"`
		Enabled *bool   `json:"enabled,omitempty"`
		Auto    *bool   `json:"auto,omitempty"`
	}
}
type importExternalOutput struct {
	Body struct {
		Imported []externalNodeView `json:"imported"`
	}
}

func (h *handlers) registerExternalNodes() {
	huma.Register(h.api, huma.Operation{OperationID: "list-external-nodes", Method: http.MethodGet, Path: "/api/v1/external-nodes", Summary: "Внешние JSON-ноды", Tags: []string{"node"}}, h.listExternalNodes)
	huma.Register(h.api, huma.Operation{OperationID: "import-external-nodes", Metadata: sessionOnly, Extensions: sessionOnlyExt, Method: http.MethodPost, Path: "/api/v1/external-nodes/import", Summary: "Импортировать Xray JSON outbounds", Tags: []string{"node"}, DefaultStatus: http.StatusCreated}, h.importExternalNodes)
	huma.Register(h.api, huma.Operation{OperationID: "patch-external-node", Metadata: sessionOnly, Extensions: sessionOnlyExt, Method: http.MethodPatch, Path: "/api/v1/external-nodes/{id}", Summary: "Изменить внешнюю ноду", Tags: []string{"node"}}, h.patchExternalNode)
	huma.Register(h.api, huma.Operation{OperationID: "delete-external-node", Metadata: sessionOnly, Extensions: sessionOnlyExt, Method: http.MethodDelete, Path: "/api/v1/external-nodes/{id}", Summary: "Удалить внешнюю ноду", Tags: []string{"node"}, DefaultStatus: http.StatusNoContent}, h.deleteExternalNode)
}

func (h *handlers) listExternalNodes(ctx context.Context, _ *struct{}) (*externalNodesOutput, error) {
	rows, err := h.d.Store.DB.QueryContext(ctx, `SELECT id,name,protocol,enabled,in_auto,mihomo->>'server',(mihomo->>'port')::int FROM external_nodes ORDER BY sort,id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := &externalNodesOutput{Body: []externalNodeView{}}
	for rows.Next() {
		var x externalNodeView
		var enabled, auto int16
		if err = rows.Scan(&x.ID, &x.Name, &x.Protocol, &enabled, &auto, &x.Server, &x.Port); err != nil {
			return nil, err
		}
		x.Enabled = enabled != 0
		x.Auto = auto != 0
		out.Body = append(out.Body, x)
	}
	return out, rows.Err()
}
func (h *handlers) importExternalNodes(ctx context.Context, in *importExternalInput) (*importExternalOutput, error) {
	imported, err := externalnodes.Import([]byte(in.Body.JSON), in.Body.Name)
	if err != nil {
		return nil, huma.Error422UnprocessableEntity("external_json_invalid", err)
	}
	now := time.Now().Unix()
	out := &importExternalOutput{}
	out.Body.Imported = []externalNodeView{}
	tx, err := h.d.Store.DB.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	for _, p := range imported {
		raw, e := json.Marshal(p.Mihomo)
		if e != nil {
			return nil, e
		}
		var id int64
		err = tx.QueryRowContext(ctx, `INSERT INTO external_nodes(name,protocol,uri,mihomo,source_json,enabled,in_auto,sort,created_at,updated_at) VALUES($1,$2,$3,$4,$5,1,1,(SELECT COALESCE(MAX(sort),0)+1 FROM external_nodes),$6,$6) RETURNING id`, p.Name, p.Protocol, p.URI, raw, in.Body.JSON, now).Scan(&id)
		if err != nil {
			return nil, err
		}
		out.Body.Imported = append(out.Body.Imported, externalNodeView{ID: id, Name: p.Name, Protocol: p.Protocol, Enabled: true, Auto: true, Server: p.Server, Port: p.Port})
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	settings.Touch()
	return out, nil
}
func (h *handlers) patchExternalNode(ctx context.Context, in *patchExternalInput) (*struct{}, error) {
	var name sql.NullString
	if in.Body.Name != nil {
		n := strings.TrimSpace(*in.Body.Name)
		if n == "" || len([]rune(n)) > 120 {
			return nil, huma.Error422UnprocessableEntity("external_name_invalid")
		}
		name = sql.NullString{String: n, Valid: true}
	}
	if in.Body.Name == nil && in.Body.Enabled == nil && in.Body.Auto == nil {
		return nil, huma.Error422UnprocessableEntity("empty_patch")
	}
	res, err := h.d.Store.DB.ExecContext(ctx, `UPDATE external_nodes SET name=COALESCE($1,name),enabled=COALESCE($2,enabled),in_auto=COALESCE($3,in_auto),updated_at=$4 WHERE id=$5`, name, in.Body.Enabled, in.Body.Auto, time.Now().Unix(), in.ID)
	if err != nil {
		return nil, err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return nil, huma.Error404NotFound("external_node_not_found")
	}
	settings.Touch()
	return nil, nil
}
func (h *handlers) deleteExternalNode(ctx context.Context, in *externalIDInput) (*struct{}, error) {
	res, err := h.d.Store.DB.ExecContext(ctx, `DELETE FROM external_nodes WHERE id=$1`, in.ID)
	if err != nil {
		return nil, err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return nil, huma.Error404NotFound("external_node_not_found")
	}
	settings.Touch()
	return nil, nil
}
