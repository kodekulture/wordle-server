# Wordle Server

## Sections

- [Logic](https://github.com/Chat-Map/wordle-server/#logic)
- [Data](https://github.com/Chat-Map/wordle-server/#data)
- [Struct](https://github.com/Chat-Map/wordle-server/#struct)
- [Endpoint](https://github.com/Chat-Map/wordle-server/#struct)
- [Websocket](https://github.com/Chat-Map/wordle-server/#websockets)
- [Future Game Modes](https://github.com/Chat-Map/wordle-server/#websockets)

## Logic 🧠

* Each event should be in a particular format, `server/<action>` and `client/<action>`. The former describes actions sent to the server, while the latter describes actions sent to the client.
* If the game has ended:
    * Frontend should display results

* Else:
  * Connect to [WS] /live
  * If the session has not ended: User has access to /play

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

### [POST] /login

* Fields
  * Username (unique)
  * Password (just put anything bro)
* Response
  * Access token

### [POST] /register

* Fields:
  * Username (unique)
  * Password (just put anything bro)
* Response
  * Access token

### [POST] /create/room 🔒

* Creates a new room returning the id of this new room

### [GET] /join/room/{id} 🔒

* Response:
  * {token: xxxx}

### [GET] /room/ 🔒

* Return list of all games played by the user

### [GET] /room/{roomID} 🔒

* Response
  * Submitted words (For this specific user)
  * All the game details (Finished, players (usernames)

## Websockets 🚀

### [WS] /live?token=xxxx

### [WS] /live?player=<playerUsername>&room=<roomID>

* When a user joins the lobby of a game
* If the game has started:
    * Fetch array of sessions for every player
    * The Player is rejected if he was not in the room before the game started
    * The player is reconnected if he is an original member of the room
* Else:
    * Player is uniquely added to the room
* client/message: `{USERNAME} has joined the lobby`

### [WSE] server/message

* Broadcasts a message to everyone in the lobby
* Fields:
    * Sender: <playerUsername>
    * Message


### [WSE] server/play

* Checks:
  * If the user has already won || The game has ended || Has finished his guesses
    * no-op

* Fields:
  * Word

* Response:
  * []int, Where each `i` can be {0,1,2,3}, Length of []int is the size of the word

### [WSE] client/result

* Returns the result of a `server/play` to show the result of a played word.

### [WSE] client/data

* Returns the current game data, it is usually sent at the beginning of a connection/reconnection to update the stored data on the client about the current game status.

### [WSE] client/start

* Notify users that the game has started, Now they can submit words using /play

### [WSE] client/play

* When a player submits a word, other users are notified about the status of the leaderboard of this user.
* Users receives a `Session` object to update leaderboard


# Future Game Modes ✨

* Sprint mode: Unlimited trials (shortest time to guess a word is only used to determine the winner of this game mode)
* Wizard mode: (the smallest trials to guess a word wins, when there is a tie, the first to get the smallest trials win).