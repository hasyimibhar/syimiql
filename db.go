package main

type DB struct {}

func Create(dir string) (*DB, error) {
	return &DB{}, nil
}

func (db *DB) Exec(stmt string) error {
	return nil
}

func (db *DB) Query(query string) (string, error) {
	return "NOP", nil
}
