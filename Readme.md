# HELM

This is meant to be a main controller for my day-to-day work and other tasks. Currently:

1. Markdown editor for a daily-status
2. Visibility of my assigned tasks/ made requests in the various work systems.~~~~

### Todo

* [ ] Add in a `@tag` search capability
  * [ ] Does basic search count?
* [x] Add in a Markdown preview 
  * [x] Fyne's Markdown preview is crap. Use another?
  * Kinda. Now just using HTML
* [x] Add in navigation by calendar dates
* [x] Add a search, integrate with Finder/ Win search?
  * Couldn't find out how to integrate, so just used `grep` and `findstr`
* [x] Add a status bar to tasks to show loading/ activity
* [x] Integration with Planner and JIRA and Cherwell for a combined task view
* [ ] Add ability to reassign jobs
* [ ] Add ability to comment on jobs
* [ ] Add ability to see current comments/ journals for jobs
* [ ] Change sort icons to be a filter to show/ hide backlog items
* [ ] Change the pull-tasks to be a complete background task rather than just the access token
  * [ ] Send to the back end agent a request for tasks rather than a request for access token
  * [ ] If the data is less than X seconds old, just return the last data.
  * [ ] Otherwise, do a read and return.
  * [ ] Only one request process per backend tool at a time