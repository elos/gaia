Gaia - Elos Ontology Server
---------------------------

Gaia is an HTTP protocol for interacting with the elos ontology. It consists of just two endpoints: `/record/` and `/record/query`.

This document serves both as a spec for the gaia http protocol, but also as a discussion for it's uses.

### '/record/`

#### GET

Conceptual: Retrieve a model by it's (kind, id) pair.

Example: GET http://gaia.elos.io/record/?kind=task&id=3


**Required** parameters: `kind` and `id`.

 * The `kind` is the model kind to retrieve
 * The `id` is the id of the model to retrieve

Succesful Responses:
 * (200, model as the payload)


Error responses:
 * (400, "You must specify a kind")
 * (400, "You must specify an id")
 * (400, "The kind is not recognized")
 * (400, "The id is invalid")
 and others

#### POST

Conceptual: Create a new model or update the included model

Example: POST http://gaia.elos.io/record/?kind=task
            {
                "name": "New Task",
            }

**Required** parameters: `kind`

The model payload may or may not have an ID listed. If it does have an id, then the record is updated, if it does not have an id the record is created, and the response body should be inspected to discover which id was assigned. Not that POSTing an update to an invalid (kind, id) pair results in a 404.

Successful Responses:
 * (200, The model was succesfully updated)
 * (201, The model was succesfully created)

Error Responses:
 * (400, "You must specify a kind")
 * (400, "The kind is not recognized")
 * (400, "The id is invalid")
 and others

#### DELETE

Conceputal: Delete the model specified by the (kind, id) pair.

Example: DELETE http://gaia.elos.io/record/?kind=task&id=4

**Required** parameters: `kind` and `id`

Succesful Response:
 * (204, The model was succesfully deleted)

Error Responses:
 * (400, "You must specify a kind")
 * (400, "You must specify an id")
 * (400, "The kind is not recognized")
 * (400, "The id is invalid")

### `/record/query`

#### POST

Conceptual: Query the elos ontology.

Example: POST http://gaia.elos.io/record/query/?kind=task
            {
                "name": "Existing Task"
            }

**Required** parameters: `kind`

The payload contains a list of data attributes to match against. Currently the elos ontology only supports the simplest of data queries, in which you retrieve the entire record, and you can only match based on equality.

Succesful Response:
    (200, "Succesfully queried - and the payload should contain some records")

Error Responses:
 * (400, "You must specify a kind")
 * (404, "No records found matching the query")
 and others


