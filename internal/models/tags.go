package models

import (
	"database/sql"
	"errors"
)

type Tag struct {
	ID    int
	Name  string
	Color string
}

func (m *PostModel) InsertTag(t Tag) (int, error) {
	stmt := "INSERT INTO tag (name, color) VALUES (?, ?)"

	result, err := m.DB.Exec(stmt, t.Name, t.Color)
	if err != nil {
		return 0, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}

	return int(id), nil
}

func (m *PostModel) InsertTagIfNotExists(t Tag) (int, error) {
	et, err := m.GetTagByName(t.Name)
	if err != nil {
		if errors.Is(err, ErrNoRecord) {
			return m.InsertTag(t)
		}

		return 0, err
	}

	return et.ID, nil
}

func (m *PostModel) InsertPostTag(t Tag) (int, error) {

}

func (m *PostModel) GetTagByName(name string) (Tag, error) {
	stmt := "SELECT id, name, color FROM tag WHERE name = ?"
	
	var t Tag

	err := m.DB.QueryRow(stmt, name).Scan(&t.ID, &t.Name, &t.Color)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Tag{}, ErrNoRecord
		} else {
			return Tag{}, err
		}
	}

	return t, nil
}

func (m *PostModel) InsertPostTagIfNotExists(p Post, t Tag) (int, error) {
	// stmt := "INSERT INTO post_tag (post_id, tag_id) VALUES (?, ?)"

	return 0, nil
}

func (m *PostModel) PostTagExists(p, t int) (bool, error) {
	stmt := "SELECT id FROM post_tag WHERE post_id = ? AND tag_id = ?"

	var id int
	err := m.DB.QueryRow(stmt, p, t).Scan(&id)
	if err != nil {
		if err != sql.ErrNoRows {
			return false, err
		}

		return false, nil
	}

	return true, nil
}

func (m *PostModel) AllTagsForPost(postId int) ([]Tag, error) {
	stmt := `SELECT t.id, t.name, t.color FROM tag t
	INNER JOIN post_tag pt ON t.id = pt.tag_id
	INNER JOIN post p ON p.id = pt.post_id
	WHERE p.id = ?`

	rows, err := m.DB.Query(stmt, postId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tags []Tag
	for rows.Next() {
		var t Tag
		err := rows.Scan(&t.ID, &t.Name, &t.Color)
		if err != nil {
			return nil, err
		}

		tags = append(tags, t)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return tags, nil
}

func (m *PostModel) AllTags() ([]Tag, error) {
	stmt := `SELECT id, name, color FROM tag`
	rows, err := m.DB.Query(stmt)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	var tags []Tag

	for rows.Next() {
		var t Tag
		err := rows.Scan(&t.ID, &t.Name, &t.Color)
		if err != nil {
			return nil, err
		}

		tags = append(tags, t)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return tags, nil
}
