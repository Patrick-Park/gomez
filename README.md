_WORK IN PROGRESS_

This package aims to become a thoroughly tested full-service mail server, containing a CLI, an SMTP, MTA, a POP3 and potentially a JSON REST API.

__SMTP Package__ - SMTP server  
__Mailbox Package__ - Data layer used by all components  
__Agent Package__ - Mail Transfer Agent (sends enqueued outbound mail)  

This package also uses [mocks](http://github.com/gbbr/mocks) for mocking network connections and addresses, and [jamon](http://github.com/gbbr/jamon) for parsing configuration files.

_(more to come)_
