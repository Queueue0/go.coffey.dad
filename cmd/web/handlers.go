package main

import (
	"errors"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"os"
	"strconv"
	"time"

	"coffey.dad/internal/models"
	"coffey.dad/internal/validator"

	"github.com/julienschmidt/httprouter"
)

func (app *application) home(w http.ResponseWriter, r *http.Request) {
	posts, err := app.posts.Latest(5)
	if err != nil {
		app.serverError(w, r, err)
		return
	}

	data := app.newTemplateData(r)
	data.Posts = posts

	app.render(w, r, http.StatusOK, "home.tmpl", data)
}

func (app *application) postView(w http.ResponseWriter, r *http.Request) {
	params := httprouter.ParamsFromContext(r.Context())

	id, err := strconv.Atoi(params.ByName("id"))
	if err != nil {
		app.notFound(w, r)
		return
	}

	post, err := app.posts.Get(id)
	if err != nil {
		if errors.Is(err, models.ErrNoRecord) {
			app.notFound(w, r)
		} else {
			app.serverError(w, r, err)
		}
		return
	}

	if post.IsDraft {
		app.notFound(w, r)
		return
	}

	post.ParseBody()

	flash := app.sessionManager.PopString(r.Context(), "flash")
	data := app.newTemplateData(r)
	data.Post = post
	data.Flash = flash

	app.render(w, r, http.StatusOK, "post_view.tmpl", data)
}

func (app *application) postList(w http.ResponseWriter, r *http.Request) {
	posts, err := app.posts.All()
	if err != nil {
		app.serverError(w, r, err)
		return
	}

	flash := app.sessionManager.PopString(r.Context(), "flash")
	data := app.newTemplateData(r)
	data.Posts = posts
	data.Flash = flash

	app.render(w, r, http.StatusOK, "post_list.tmpl", data)
}

func (app *application) draftList(w http.ResponseWriter, r *http.Request) {
	drafts, err := app.posts.AllDrafts()
	if err != nil {
		app.serverError(w, r, err)
		return
	}

	flash := app.sessionManager.PopString(r.Context(), "flash")
	data := app.newTemplateData(r)
	data.Drafts = drafts
	data.Flash = flash

	app.render(w, r, http.StatusOK, "draft_list.tmpl", data)
}

type postForm struct {
	Title string
	Body  string
	validator.Validator
}

func (app *application) newPost(w http.ResponseWriter, r *http.Request) {
	form := postForm{}
	data := app.newTemplateData(r)
	data.NewPost = true
	data.Form = form

	app.render(w, r, http.StatusOK, "add_edit_post.tmpl", data)
}

func (app *application) newPostSubmit(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	form := postForm{
		Title: r.PostForm.Get("title"),
		Body:  r.PostForm.Get("body"),
	}

	asDraft, err := strconv.ParseBool(r.PostForm.Get("asDraft"))
	if err != nil {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	form.CheckField(validator.NotBlank(form.Title), "title", "This field cannot be blank")
	form.CheckField(validator.MaxChars(form.Title, 256), "title", "This field cannot be longer than 256 characters")
	form.CheckField(validator.NotBlank(form.Body), "body", "This field cannot be blank")

	if !form.Valid() {
		data := app.newTemplateData(r)
		data.Form = form
		data.NewPost = true
		app.render(w, r, http.StatusUnprocessableEntity, "add_edit_post.tmpl", data)
		return
	}

	now := time.Now()
	id, err := app.posts.Insert(models.Post{
		Title: form.Title,
		Body: template.HTML(form.Body),
		IsDraft: asDraft,
		Created: now,
		Modified: now,
	})

	if err != nil {
		app.serverError(w, r, err)
		return
	}

	if !asDraft {
		app.sessionManager.Put(r.Context(), "flash", "Published successfully!")
		http.Redirect(w, r, fmt.Sprintf("/blog/post/%d", id), http.StatusSeeOther)
	} else {
		app.sessionManager.Put(r.Context(), "flash", "Draft saved successfully!")
		http.Redirect(w, r, "/blog/drafts", http.StatusSeeOther)
	}
}

func (app *application) editPost(w http.ResponseWriter, r *http.Request) {
	params := httprouter.ParamsFromContext(r.Context())

	id, err := strconv.Atoi(params.ByName("id"))
	if err != nil {
		app.notFound(w, r)
		return
	}

	post, err := app.posts.Get(id)
	if err != nil {
		if errors.Is(err, models.ErrNoRecord) {
			app.notFound(w, r)
		} else {
			app.serverError(w, r, err)
		}
		return
	}

	form := postForm{
		Title: post.Title,
		Body:  string(post.Body),
	}

	data := app.newTemplateData(r)
	data.Post = post
	data.Form = form
	data.NewPost = false

	app.render(w, r, http.StatusOK, "add_edit_post.tmpl", data)
}

func (app *application) editPostSubmit(w http.ResponseWriter, r *http.Request) {
	params := httprouter.ParamsFromContext(r.Context())

	id, err := strconv.Atoi(params.ByName("id"))
	if err != nil {
		app.notFound(w, r)
		return
	}

	err = r.ParseForm()
	if err != nil {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	form := postForm{
		Title: r.PostForm.Get("title"),
		Body:  r.PostForm.Get("body"),
	}

	asDraft, err := strconv.ParseBool(r.PostForm.Get("asDraft"))
	if err != nil {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	form.CheckField(validator.NotBlank(form.Title), "title", "This field cannot be blank")
	form.CheckField(validator.MaxChars(form.Title, 256), "title", "This field cannot be longer than 256 characters")
	form.CheckField(validator.NotBlank(form.Body), "body", "This field cannot be blank")

	if !form.Valid() {
		data := app.newTemplateData(r)
		data.Form = form
		data.NewPost = false
		app.render(w, r, http.StatusUnprocessableEntity, "add_edit_post.tmpl", data)
		return
	}

	p, err := app.posts.Get(id)
	if err != nil {
		app.serverError(w, r, err)
		return
	}

	p.Title = form.Title
	p.Body = template.HTML(form.Body)
	p.IsDraft = asDraft
	p.Modified = time.Now()

	err = app.posts.Update(p)
	if err != nil {
		app.serverError(w, r, err)
		return
	}

	if !asDraft {
		http.Redirect(w, r, fmt.Sprintf("/blog/post/%d", id), http.StatusSeeOther)
	} else {
		http.Redirect(w, r, "/blog/drafts", http.StatusSeeOther)
	}
}

func (app *application) editDraft(w http.ResponseWriter, r *http.Request) {
	params := httprouter.ParamsFromContext(r.Context())

	id, err := strconv.Atoi(params.ByName("id"))
	if err != nil {
		app.notFound(w, r)
		return
	}

	draft, err := app.posts.Get(id)
	if err != nil {
		if errors.Is(err, models.ErrNoRecord) {
			app.notFound(w, r)
		} else {
			app.serverError(w, r, err)
		}
		return
	}

	form := postForm{
		Title: draft.Title,
		Body:  string(draft.Body),
	}

	data := app.newTemplateData(r)
	data.Post = draft
	data.Form = form
	data.NewPost = false

	app.render(w, r, http.StatusOK, "edit_draft.tmpl", data)
}

func (app *application) editDraftSubmit(w http.ResponseWriter, r *http.Request) {
	params := httprouter.ParamsFromContext(r.Context())

	id, err := strconv.Atoi(params.ByName("id"))
	if err != nil {
		app.notFound(w, r)
		return
	}

	err = r.ParseForm()
	if err != nil {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	form := postForm{
		Title: r.PostForm.Get("title"),
		Body:  r.PostForm.Get("body"),
	}

	asDraft, err := strconv.ParseBool(r.PostForm.Get("asDraft"))
	if err != nil {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	form.CheckField(validator.NotBlank(form.Title), "title", "This field cannot be blank")
	form.CheckField(validator.MaxChars(form.Title, 256), "title", "This field cannot be longer than 256 characters")
	form.CheckField(validator.NotBlank(form.Body), "body", "This field cannot be blank")

	if !form.Valid() {
		data := app.newTemplateData(r)
		data.Form = form
		app.render(w, r, http.StatusUnprocessableEntity, "edit_draft.tmpl", data)
		return
	}

	p, err := app.posts.Get(id)
	if err != nil {
		app.serverError(w, r, err)
		return
	}

	p.Title = form.Title
	p.Body = template.HTML(form.Body)
	p.IsDraft = asDraft

	if !p.IsDraft {
		p.Created = time.Now()
		p.Modified = p.Created
	}
	err = app.posts.Update(p)

	if asDraft {
		http.Redirect(w, r, "/blog/drafts", http.StatusSeeOther)
	} else {
		app.sessionManager.Put(r.Context(), "flash", "Published successfully!")
		http.Redirect(w, r, fmt.Sprintf("/blog/post/%d", id), http.StatusSeeOther)
	}
}

type loginForm struct {
	Username string
	Password string
	validator.Validator
}

func (app *application) login(w http.ResponseWriter, r *http.Request) {
	data := app.newTemplateData(r)
	data.Form = loginForm{}
	app.render(w, r, http.StatusOK, "login.tmpl", data)
}

func (app *application) loginSubmit(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	form := loginForm{
		Username: r.PostForm.Get("username"),
		Password: r.PostForm.Get("password"),
	}

	form.CheckField(validator.NotBlank(form.Username), "username", "This field cannot be blank")
	form.CheckField(validator.NotBlank(form.Password), "password", "Thid field cannot be blank")

	if !form.Valid() {
		data := app.newTemplateData(r)
		data.Form = form
		app.render(w, r, http.StatusUnprocessableEntity, "login.tmpl", data)
		return
	}

	id, err := app.users.Authenticate(form.Username, form.Password)
	if err != nil {
		if errors.Is(err, models.ErrInvalidCredentials) {
			form.AddNonFieldError("Email or password is incorrect")

			data := app.newTemplateData(r)
			data.Form = form
			app.render(w, r, http.StatusUnprocessableEntity, "login.tmpl", data)
		} else {
			app.serverError(w, r, err)
		}
		return
	}

	err = app.sessionManager.RenewToken(r.Context())
	if err != nil {
		app.serverError(w, r, err)
		return
	}

	app.sessionManager.Put(r.Context(), "authenticatedUserID", id)
	redirect := app.sessionManager.PopString(r.Context(), "redirectPathAfterLogin")
	if redirect == "" {
		redirect = "/"
	}

	http.Redirect(w, r, redirect, http.StatusSeeOther)
}

func (app *application) logoutSubmit(w http.ResponseWriter, r *http.Request) {
	err := app.sessionManager.RenewToken(r.Context())
	if err != nil {
		app.serverError(w, r, err)
	}

	app.sessionManager.Remove(r.Context(), "authenticatedUserID")

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

type uploadForm struct {
	validator.Validator
}

func (app *application) upload(w http.ResponseWriter, r *http.Request) {
	data := app.newTemplateData(r)
	data.Form = uploadForm{}

	app.render(w, r, http.StatusOK, "upload.tmpl", data)
}

func (app *application) uploadPost(w http.ResponseWriter, r *http.Request) {
	form := uploadForm{}

	err := r.ParseMultipartForm(32 << 20)
	if err != nil {
		form.AddNonFieldError("Error Uploading Image")

		data := app.newTemplateData(r)
		data.Form = form
		app.render(w, r, http.StatusInternalServerError, "upload.tmpl", data)
		return
	}

	files := r.MultipartForm.File["file"]

	for _, fileHeader := range files {
		file, err := fileHeader.Open()
		if err != nil {
			form.AddNonFieldError("Error Uploading Image")

			data := app.newTemplateData(r)
			data.Form = form
			app.render(w, r, http.StatusInternalServerError, "upload.tmpl", data)
			return
		}

		defer file.Close()

		buff := make([]byte, 512)
		_, err = file.Read(buff)
		if err != nil {
			form.AddNonFieldError("Error Uploading Image")

			data := app.newTemplateData(r)
			data.Form = form
			app.render(w, r, http.StatusInternalServerError, "upload.tmpl", data)
			return
		}

		_, err = file.Seek(0, io.SeekStart)
		if err != nil {
			form.AddNonFieldError("Error Uploading Image")

			data := app.newTemplateData(r)
			data.Form = form
			app.render(w, r, http.StatusInternalServerError, "upload.tmpl", data)
			return
		}

		err = os.MkdirAll("./uploads", os.ModePerm)
		if err != nil {
			form.AddNonFieldError("Error Uploading Image")

			data := app.newTemplateData(r)
			data.Form = form
			app.render(w, r, http.StatusInternalServerError, "upload.tmpl", data)
			return
		}

		f, err := os.Create(fmt.Sprintf("./uploads/%s", fileHeader.Filename))
		if err != nil {
			form.AddNonFieldError("Error Uploading Image")

			data := app.newTemplateData(r)
			data.Form = form
			app.render(w, r, http.StatusInternalServerError, "upload.tmpl", data)
			return
		}

		defer f.Close()

		_, err = io.Copy(f, file)
		if err != nil {
			form.AddNonFieldError("Error Uploading Image")

			data := app.newTemplateData(r)
			data.Form = form
			app.render(w, r, http.StatusInternalServerError, "upload.tmpl", data)
			return
		}
	}

	data := app.newTemplateData(r)
	data.Form = form
	data.Flash = "Image uploaded successfully!"
	app.render(w, r, http.StatusOK, "upload.tmpl", data)
}

func (app *application) imagePicker(w http.ResponseWriter, r *http.Request) {
	var fileNames []string
	files, err := os.ReadDir("./uploads/")
	if err != nil {
		app.serverError(w, r, err)
		return
	}

	for _, file := range files {
		if !file.IsDir() {
			fileNames = append(fileNames, file.Name())
		}
	}

	data := app.newTemplateData(r)
	data.FileNames = fileNames

	app.render(w, r, http.StatusOK, "imagepicker.tmpl", data)
}
