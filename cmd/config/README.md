# Eolymp

`apiurl` - the api link to Eolymp, you are not supposed to change it

`username` - the username of your Eolymp account (this account should have access to the space to which you are going to upload problems)

`password` - the password of your Eolymp account

`spaceimport` - the space ID of the space **FROM** which you want to upload problems. If you want to upload problems from some other source, for example, Polygon, you don't have to fill it out

# Polygon

You should fill these field out only if you want to upload problems from Polygon

`login` - the username of your Polygon account

`password` - the password of your Polygon account

# Telegram

You should fill these field out only if you want to run telegram bot

`token` - the token of your Telegram bot

`chatid` - the chat ID of your Telegram chat, the bot should have access to send messages to that chat

`problems` - you should fill out these fields for each problem

- `id` - the ID for internal use. For example, `A`, `B`, `C`. Those lettes you will need to type in the Telegram chat in order to upload the problem
- `link` - the link to the Polygon problem. If should have the following format `https://polygon.codeforces.com/20wkaGA/arsijo/nameoftheproblem`
- `pid` - the ID of the problem. Please note that it is NOT a number in the list of the problems. In order to get the ID of the problem, you should click on the problem and you will be the ID in the link. New problems have 5 digits at the moment of writing

# General

`source` - this data will be used to note the source of the problem in the Eolymp. You can type anything you want or leave it empty. For example, you can type `UOI 2023`.

`spaceid` - the space ID of the space to which you want to upload problems
