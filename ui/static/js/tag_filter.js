document.addEventListener("DOMContentLoaded", () => {
  const filterParam = new URLSearchParams(window.location.search).get('filter');
  const filterLink = document.getElementById("filterLink");
  const tagFilter = document.getElementById("tagFilter");
  let tagText = tagFilter.value;

  if (decodeURIComponent(tagText) == filterParam) {
    filterLink.classList.remove("is-info");
    filterLink.classList.add("is-danger");
    filterLink.setAttribute("href", "/blog");
    filterLink.innerHTML = "Clear";
  }

  tagFilter.addEventListener("change", () => {
    if (filterLink.classList.contains("is-danger")) {
      filterLink.classList.remove("is-danger");
      filterLink.classList.add("is-info");
      filterLink.innerHTML = "Filter";
    }
    tagText = tagFilter.value
    if (tagText) {
      filterLink.setAttribute("href", "/blog?filter="+tagText);
    } else {
      filterLink.setAttribute("href", "/blog");
    }
  });
});
