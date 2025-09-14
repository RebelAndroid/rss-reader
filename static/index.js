async function mark_as_read(event, url) {
    let response = await fetch("/api/mark_read", {method: "POST", body: url})
    console.log(event.target.closest('button').parentElement.parentElement)
    event.target.closest('button').parentElement.parentElement.remove()
}