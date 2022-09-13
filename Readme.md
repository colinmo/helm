# HELM

This is meant to be a main controller for my day-to-day work and other tasks. Currently:

1. Markdown editor for a daily-status
2. Deactivate parental controls on my internet for a set period of time

### Todo

* [ ] Add in a `@tag` search capability
  * [ ] Does basic search count?
* [x] Add in a Markdown preview 
  * [x] Fyne's Markdown preview is crap. Use another?
  * Kinda. Now just using HTML
* [x] Add in navigation by calendar dates
* [x] Add a search, integrate with Finder/ Win search?
  * Couldn't find out how to integrate, so just used `grep` and `findstr`
* [ ] Add a status bar to tasks to show loading/ activity
* [ ] Integration with Planner and JIRA and Cherwell for a combined task view


### Extra

I've been assigned a task in our Dynamics Project interface. _SIGH_.

To get my projects

#### Authenticate - OAuth

| Field             | Value                                                                                                        |
|-------------------|--------------------------------------------------------------------------------------------------------------|
| grant_type        | Implicit                                                                                                     |
| authorization_url | https://login.microsoftonline.com/common/oauth2/authorize?resource=https://orgb972b9ec.api.crm6.dynamics.com |
| client_id         | 51f81489-12ee-4a9e-aaae-a2591f45987d                                                                         |
| redirect_uri      | https://localhost/                                                                                           |
| response_type     | access_token                                                                                                 |

#### Get my Project Tasks

```
curl --request GET \
  --url 'https://orgb972b9ec.api.crm6.dynamics.com/api/data/v9.2/msdyn_projecttasks?savedQuery=4792d21c-d6b4-e511-80e4-00155db8d81d&%24select=msdyn_subject%2C_ownerid_value%2Cmsdyn_priority' \
  --header 'Authorization: Bearer x' \
  --header 'Content-Type: application/json'
```

#### Get project task information