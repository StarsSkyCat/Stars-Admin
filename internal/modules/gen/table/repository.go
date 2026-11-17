package table

import (
	"context"

	"github.com/StarsSkyCat/Stars-Admin/internal/core/model"
	"github.com/StarsSkyCat/Stars-Admin/internal/core/response"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type repository struct{ db *gorm.DB }

func (r repository) conn(ctx context.Context, tx *gorm.DB) *gorm.DB {
	if tx == nil {
		tx = r.db
	}
	return tx.WithContext(ctx)
}
func (r repository) dbTables(ctx context.Context, schema string) ([]DBTable, error) {
	rows := []DBTable{} // PostgreSQL catalog 是生成器登记例外；schema 是参数，禁止拼接标识符。
	err := r.conn(ctx, nil).Raw(`SELECT c.relname AS name,COALESCE(obj_description(c.oid,'pg_class'),'') AS comment FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace WHERE n.nspname=? AND c.relkind='r' AND NOT c.relispartition ORDER BY c.relname`, schema).Scan(&rows).Error
	return rows, err
}
func (r repository) describe(ctx context.Context, tx *gorm.DB, schema, name string) (Snapshot, error) {
	db := r.conn(ctx, tx)
	out := Snapshot{Name: name, Columns: []MetadataColumn{}, Constraints: []string{}, Indexes: []string{}}
	var heads []DBTable // 表名作为筛选值参数化；所有 PG catalog SQL 收敛于本仓储。
	if err := db.Raw(`SELECT c.relname AS name,COALESCE(obj_description(c.oid,'pg_class'),'') AS comment FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace WHERE n.nspname=? AND c.relname=? AND c.relkind='r' AND NOT c.relispartition`, schema, name).Scan(&heads).Error; err != nil {
		return out, err
	}
	if len(heads) != 1 {
		return out, response.ErrNotFound
	}
	out.Comment = heads[0].Comment
	err := db.Raw(`SELECT a.attname AS name,COALESCE(col_description(c.oid,a.attnum),'') AS comment,format_type(a.atttypid,a.atttypmod) AS db_type,t.typname AS udt,NOT a.attnotnull AS nullable,i.character_maximum_length AS max_length,i.numeric_precision AS precision,i.numeric_scale AS scale,i.column_default AS "default",EXISTS(SELECT 1 FROM pg_constraint p WHERE p.conrelid=c.oid AND p.contype='p' AND a.attnum=ANY(p.conkey)) AS "primary",a.attidentity<>'' AS identity,a.attgenerated<>'' AS generated,a.attnum AS position FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace JOIN pg_attribute a ON a.attrelid=c.oid AND a.attnum>0 AND NOT a.attisdropped JOIN pg_type t ON t.oid=a.atttypid JOIN information_schema.columns i ON i.table_schema=n.nspname AND i.table_name=c.relname AND i.column_name=a.attname WHERE n.nspname=? AND c.relname=? ORDER BY a.attnum`, schema, name).Scan(&out.Columns).Error
	if err != nil {
		return out, err
	}
	if err := db.Raw(`SELECT quote_ident(p.conname)||' '||pg_get_constraintdef(p.oid) FROM pg_constraint p JOIN pg_class c ON c.oid=p.conrelid JOIN pg_namespace n ON n.oid=c.relnamespace WHERE n.nspname=? AND c.relname=? ORDER BY p.conname`, schema, name).Scan(&out.Constraints).Error; err != nil {
		return out, err
	}
	err = db.Raw(`SELECT pg_get_indexdef(i.indexrelid) FROM pg_index i JOIN pg_class c ON c.oid=i.indrelid JOIN pg_namespace n ON n.oid=c.relnamespace WHERE n.nspname=? AND c.relname=? AND NOT EXISTS(SELECT 1 FROM pg_constraint p WHERE p.conindid=i.indexrelid) ORDER BY i.indexrelid`, schema, name).Scan(&out.Indexes).Error
	return out, err
}
func (r repository) get(ctx context.Context, tx *gorm.DB, id model.ID) (table, error) {
	var row table
	err := r.conn(ctx, tx).Clauses(clause.Locking{Strength: "UPDATE"}).First(&row, "id = ?", id).Error
	return row, err
}
func (r repository) columns(ctx context.Context, tx *gorm.DB, id model.ID, all bool) ([]column, error) {
	db := r.conn(ctx, tx)
	if all {
		db = db.Unscoped()
	}
	rows := []column{}
	err := db.Where("table_id = ?", id).Order("sort,id").Find(&rows).Error
	return rows, err
}
func (r repository) save(ctx context.Context, tx *gorm.DB, row *table, create bool) error {
	if create {
		return r.conn(ctx, tx).Create(row).Error
	}
	return r.conn(ctx, tx).Save(row).Error
}
func (r repository) saveColumn(ctx context.Context, tx *gorm.DB, row *column, create bool) error {
	if create {
		return r.conn(ctx, tx).Create(row).Error
	}
	return r.conn(ctx, tx).Unscoped().Save(row).Error
}
func (r repository) list(ctx context.Context, q response.PageQuery) ([]table, int64, error) {
	db := r.conn(ctx, nil).Model(&table{})
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	rows := []table{}
	err := db.Order("id").Offset((q.Page - 1) * q.PageSize).Limit(q.PageSize).Find(&rows).Error
	return rows, total, err
}
func (r repository) remove(ctx context.Context, tx *gorm.DB, ids []model.ID, p model.ID) error {
	db := r.conn(ctx, tx)
	if err := db.Model(&column{}).Where("table_id IN ?", ids).Update("updated_by", p).Error; err != nil {
		return err
	}
	if err := db.Delete(&column{}, "table_id IN ?", ids).Error; err != nil {
		return err
	}
	if err := db.Model(&table{}).Where("id IN ?", ids).Update("updated_by", p).Error; err != nil {
		return err
	}
	return db.Delete(&table{}, "id IN ?", ids).Error
}
