# Wordle Server

## Sections

- [Logic](#logic-)
- [Data](#data-)
- [Struct](#struct-)
- [Endpoint](#struct-)
- [Websocket](#websockets-)
- [Future Game Modes](#websockets-)

## Logic 🧠

* Each event should be in a particular format, `server/<action>` and `client/<action>`. The former describes actions sent to the server, while the latter describes actions sent to the client.
* If the game has ended:
    * Frontend should display results

* Else:
  * Connect to [[WS] /live](#ws-liveplayerroom)
  * If the session has not ended: User has access to [server/play](#wse-serverplay)

## Data 🗃

* The game requires two memory areas. (Permanent) and (Temporary)

### Permanent (SQL)

* The permanent area stores the list of players and the list of games / rooms.
* Rooms can only write twice to the permanent area (beginning – to fixate the users playing this game and at the end- to store the final results of the game).
* When the results of a game are written to the permanent area, it becomes Read ONLY


### Temporary (In-memory/NoSQL)

* It is a hot write area i.e. number of writes >> number of reads.
* It stores the ongoing game session
* It performs the write to the permanent storage when the game has ended.


## Struct 💾

* **Session**: contains the progress for a single user in a game (session ended means I am done with the current game, **BUT **others might still be playing)
    * Username
    * Max Guessed letters >
    * Used trails >
    * Submission time of max Guessed letters
* **Game**: contains the progress of everyone in the game

## Endpoints 🌐

### [POST] /login 🚪

* Login to an existing user

<details open>
<summary>Fields</summary>

```json
{
  "username": "username", 
  "password": "password" 
}
```
</details>


<details open>
<summary>Response</summary>

```json
{
  "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2Vybm"
}
```

</details>

### [POST] /register 📝

* Creates a new user

<details open>
<summary>Fields</summary>

```json
{
  "username": "username", 
  "password": "password"
}
```
</details>


<details open>
<summary>Response</summary>

```json
{
  "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2Vybm"
}
```

</details>

### [POST] /create/room 🔒

* Creates a new room returning the id of this new room

<details open>
<summary>Response</summary>

```json
{
  "id": "58dbe7f6-9d5c-4d48-8eac-73db92d4437d"
}
```

</details>

### [GET] /join/room/{id} 🔒

* Creates a new unique token for this (user & room)
* Notice that the token is the same for each(user & room) pair, so requesting a token for the same (user & room) pair will return the same token.

<details open>
<summary>Response</summary>

```json
{
  "token": "cb6fddb2f88acfcbbdc6c9900510"
}
```

</details>

### [GET] /room/ 🔒

* Return list of all games played by the user

<details open>
<summary>Response</summary>

```json
[
  {
    "created_at": "2023-06-19T19:51:58.802+03:00",
    "started_at": "2023-06-19T19:53:02.886447+03:00",
    "ended_at": "2023-06-19T19:51:58.802+03:00",
    "creator": "username",
    "correct_word": "FOLKS",
    "id": "58dbe7f6-9d5c-4d48-8eac-73db92d4437d"
  },
  ...
]
```

</details>

### [GET] /room/{roomID} 🔒

* Return all the information about a specific game

<details open>
<summary>Response</summary>

```json
{
    "created_at": "2023-06-19T19:51:58.802+03:00",
    "started_at": "2023-06-19T19:53:02.886447+03:00",
    "ended_at": "2023-06-19T19:51:58.802+03:00",
    "creator": "username",
    "correct_word": "FOLKS",
    "guesses": [
        {
            "word": "FOLKS",
            "played_at": "2023-06-19T16:53:27.581099801Z",
            "status": [3,3,3,3,3]
        },
        ...
    ],
    "game_performance": [
        {
            "rank": 0,
            "username": "escalopa",
            "best": {
                "played_at": "2023-06-19T16:53:27.581099801Z",
                "status": [3,3,3,3,3]
            },
            "words_played": 3,
        },
        ...
    ],
    "id": "58dbe7f6-9d5c-4d48-8eac-73db92d4437d"
}
```

</details>

## Websockets 🚀

### [WS] /live?token=XXXXX

* Connects to the game's room
* The token provied can be obtained from the [join room endpoint](#get-joinroomid-)
* Once connected you will be able to send and receive messages from the server, messages  have two types
  * [WSE] `server/xxx` means `client` => `server`
  * [WSE] `client/xxx` means `server` => `client`

* Requests object struct
```json
{
  "event": "event_name",
  "data": "object(can be anything)", 
}
```

### [WSE] server/message
* Broadcasts a message to everyone in the lobby

<details open>
<summary>Fields</summary>

```json
{
  "data": "Hello World"
}
```
</details>

* Triggers:
  * [client/message](#wse-clientmessage)

### [WSE] client/message

* Server sends a message to all clients in the lobby (client should listen to this event to update the message box)

<details open>
<summary>Fields</summary>

```json
{
  "data": "Hello World",
  "from": "username"
}
```
</details>

### [WSE] server/play

* Sends a word to the server to be played

<details open>
<summary>Fields</summary>

```json
{
  "event": "server/play",
  "data": "FOLKS"
}
```
</details>

* Tiggers:
  * [client/play](#wse-clientplay)

### [WSE] client/play

* When a player submits a word, other users are notified about the status of the leaderboard after the user's attempt.
* The `status` is an array of 5 numbers, each number represents the status of the letter in the same position in the word.
  * `3` => correct letter and position
  * `2` => correct letter but wrong position
  * `1` => wrong letter

<details open>
<summary>Fields</summary>

```json
{
    "event": "client/play",
    "data": {
        "rank_offset": 0,
        "result": {
            "played_at": "2023-06-19T19:16:36.715290087Z",
            "status": [1,2,2,1,3] 
        },
        "leaderboard": [
          {
            "rank": 0,
            "best": [3,3,3,1,3],
            "username": "other",
            "words_played": 2
          },
          ...
        ]
    },
    "from": "escalopa"
}
```
</details>

### [WSE] server/start

* Send a signal to mark the game as started and the server should now notify other players in the game about the event.

<details open>
<summary>Fields</summary>

```json
{
  "event": "server/start",
}
```
</details>

* Triggers:
  * [client/start](#wse-clientstart)

### [WSE] client/start

* Notify users that the game has started, Now they can submit words using /play

<details open>
<summary>Fields</summary>

```json
{
  "event": "server/start",
  "data": "Game has started"",
}
```
</details>

### [WSE] client/data

* Returns the current game data, it is sent to the user when
  * The user joins the game
  * The game is started

<details open>
<summary>Fields</summary>

```json
{
    "event": "client/data",
    "data": {
        "guesses": [
            {
                "word": "FOLKS",
                "played_at": "2023-06-19T19:16:36.715290087Z",
                "status": [1,2,2,1,3]
            }
        ],
        "active": true,
        "leaderboard": [
            ...
        ]
    },
    "from": "" 
}
```
</details>

# Future Game Modes ✨

* Sprint mode: Unlimited trials (shortest time to guess a word is only used to determine the winner of this game mode)
* Wizard mode: (the smallest trials to guess a word wins, when there is a tie, the first to get the smallest trials win).