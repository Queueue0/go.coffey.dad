package models

type Scanner interface {
	Scan(dest ...any) error
}
