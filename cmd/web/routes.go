package main

import (
	"net/http"

	"coffey.dad/ui"

	"github.com/julienschmidt/httprouter"
	"github.com/justinas/alice"
)

func (app *application) routes() http.Handler {
	router := httprouter.New()

	router.NotFound = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		app.notFound(w, r)
	})

	fileServer := http.FileServer(http.FS(ui.Files))
	router.Handler(http.MethodGet, "/static/*filepath", fileServer)

	uploadServer := http.FileServer(http.Dir("./uploads"))
	router.Handler(http.MethodGet, "/uploads/*filepath", http.StripPrefix("/uploads", uploadServer))

	dynamic := alice.New(app.sessionManager.LoadAndSave, noSurf, app.authenticate)

	router.Handler(http.MethodGet, "/", dynamic.ThenFunc(app.home))
	router.Handler(http.MethodGet, "/blog", dynamic.ThenFunc(app.postList))
	router.Handler(http.MethodGet, "/blog/post/:id", dynamic.ThenFunc(app.postView))
	router.Handler(http.MethodGet, "/login", dynamic.ThenFunc(app.login))
	router.Handler(http.MethodPost, "/login", dynamic.ThenFunc(app.loginSubmit))

	protected := dynamic.Append(app.requireAuthentication)

	router.Handler(http.MethodPost, "/logout", protected.ThenFunc(app.logoutSubmit))
	router.Handler(http.MethodGet, "/blog/new", protected.ThenFunc(app.newPost))
	router.Handler(http.MethodPost, "/blog/new", protected.ThenFunc(app.newPostSubmit))
	router.Handler(http.MethodGet, "/blog/edit/:id", protected.ThenFunc(app.editPost))
	router.Handler(http.MethodPost, "/blog/edit/:id", protected.ThenFunc(app.editPostSubmit))
	router.Handler(http.MethodGet, "/upload", protected.ThenFunc(app.upload))
	router.Handler(http.MethodPost, "/upload", protected.ThenFunc(app.uploadPost))
	router.Handler(http.MethodGet, "/choose-image", protected.ThenFunc(app.imagePicker))
	router.Handler(http.MethodGet, "/choose-header-image", protected.ThenFunc(app.headerImagePicker))
	router.Handler(http.MethodGet, "/blog/drafts", protected.ThenFunc(app.draftList))
	router.Handler(http.MethodGet, "/blog/drafts/edit/:id", protected.ThenFunc(app.editPost))
	router.Handler(http.MethodPost, "/blog/drafts/edit/:id", protected.ThenFunc(app.editPostSubmit))

	standard := alice.New(app.recoverPanic, app.logRequest, secureHeaders)

	return standard.Then(router)
}
