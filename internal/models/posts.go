package models

import (
	"database/sql"
	"errors"
	"html/template"
	"time"

	"github.com/gomarkdown/markdown"
	"github.com/gomarkdown/markdown/html"
	"github.com/gomarkdown/markdown/parser"
)

type Post struct {
	ID       int
	Title    string
	Body     template.HTML
	Created  time.Time
	Modified time.Time
}

type Draft struct {
	ID    int
	Title string
	Body  template.HTML
}

type PostModel struct {
	DB *sql.DB
}

func (p *Post) ParseBody() {
	body := []byte(p.Body)
	extensions := parser.CommonExtensions | parser.AutoHeadingIDs | parser.NoEmptyLineBeforeBlock
	pr := parser.NewWithExtensions(extensions)
	doc := pr.Parse(body)

	htmlFlags := html.CommonFlags
	opts := html.RendererOptions{Flags: htmlFlags}
	renderer := html.NewRenderer(opts)

	p.Body = template.HTML(markdown.Render(doc, renderer))
}

func (d *Draft) ParseBody() {
	body := []byte(d.Body)
	extensions := parser.CommonExtensions | parser.AutoHeadingIDs | parser.NoEmptyLineBeforeBlock
	pr := parser.NewWithExtensions(extensions)
	doc := pr.Parse(body)

	htmlFlags := html.CommonFlags
	opts := html.RendererOptions{Flags: htmlFlags}
	renderer := html.NewRenderer(opts)

	d.Body = template.HTML(markdown.Render(doc, renderer))
}

func (m *PostModel) Insert(title, body string) (int, error) {
	stmt := "INSERT INTO post (title, body) VALUES (?, ?)"

	result, err := m.DB.Exec(stmt, title, body)
	if err != nil {
		return 0, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}

	return int(id), nil
}

func (m *PostModel) Update(id int, title, body string) error {
	stmt := "UPDATE post SET title=?, body=? WHERE id=?"

	result, err := m.DB.Exec(stmt, title, body, id)
	if err != nil {
		return err
	}

	_, err = result.RowsAffected()
	if err != nil {
		return err
	}

	return nil
}

func (m *PostModel) Get(id int) (Post, error) {
	var p Post

	stmt := `SELECT id, title, body, created, modified FROM post
	WHERE id=?`

	err := m.DB.QueryRow(stmt, id).Scan(&p.ID, &p.Title, &p.Body, &p.Created, &p.Modified)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Post{}, ErrNoRecord
		} else {
			return Post{}, err
		}
	}

	return p, nil
}

func (m *PostModel) Latest(n int) ([]Post, error) {
	stmt := `SELECT id, title, body, created, modified FROM post
	ORDER BY id DESC LIMIT ?`

	rows, err := m.DB.Query(stmt, n)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	var posts []Post

	for rows.Next() {
		var p Post

		err := rows.Scan(&p.ID, &p.Title, &p.Body, &p.Created, &p.Modified)
		if err != nil {
			return nil, err
		}

		p.ParseBody()

		posts = append(posts, p)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return posts, nil
}

func (m *PostModel) All() ([]Post, error) {
	stmt := `SELECT id, title, body, created, modified FROM post
	ORDER BY id DESC`

	rows, err := m.DB.Query(stmt)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	var posts []Post

	for rows.Next() {
		var p Post

		err := rows.Scan(&p.ID, &p.Title, &p.Body, &p.Created, &p.Modified)
		if err != nil {
			return nil, err
		}

		p.ParseBody()

		posts = append(posts, p)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return posts, nil
}

func (m *PostModel) InsertAsDraft(title, body string) (int, error) {
	stmt := "INSERT INTO draft (title, body) VALUES (?, ?)"

	result, err := m.DB.Exec(stmt, title, body)
	if err != nil {
		return 0, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}

	return int(id), nil
}

func (m *PostModel) PublishDraft(id int, title, body string) (int, error) {
	stmt := "INSERT INTO post (title, body) VALUES (?, ?)"

	result, err := m.DB.Exec(stmt, title, body)
	if err != nil {
		return 0, err
	}

	postId, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}

	stmt = "DELETE FROM draft WHERE id=?"
	result, err = m.DB.Exec(stmt, id)
	if err != nil {
		return 0, err
	}

	return int(postId), nil
}

func (m *PostModel) UpdateDraft(id int, title, body string) error {
	stmt := "UPDATE draft SET title=?, body=? WHERE id=?"

	result, err := m.DB.Exec(stmt, title, body, id)
	if err != nil {
		return err
	}

	_, err = result.RowsAffected()
	if err != nil {
		return err
	}

	return nil
}

func (m *PostModel) GetDraft(id int) (Draft, error) {
	var d Draft

	stmt := `SELECT id, title, body FROM draft
	WHERE id=?`

	err := m.DB.QueryRow(stmt, id).Scan(&d.ID, &d.Title, &d.Body)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Draft{}, ErrNoRecord
		} else {
			return Draft{}, err
		}
	}

	return d, nil
}

func (m *PostModel) AllDrafts() ([]Draft, error) {
	stmt := `SELECT id, title, body FROM draft
	ORDER BY id DESC`

	rows, err := m.DB.Query(stmt)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	var drafts []Draft

	for rows.Next() {
		var d Draft

		err := rows.Scan(&d.ID, &d.Title, &d.Body)
		if err != nil {
			return nil, err
		}

		d.ParseBody()

		drafts = append(drafts, d)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return drafts, nil
}
