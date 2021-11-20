===================================
Common Controls Library for Telebot
===================================

Library provides common controls to use with Telebot_ library.

Controls:

* Picklist - add inline keyboard to bots messages.
* Post Buttons - add buttons to your channel posts.
* Rating - rating buttons for channel posts.
* Keyboard - a convenient way to create a keyboard.
* Input - ask user for input and process the answer in OnText.

Abstractions:

* Form (combines other controls into a pipeline, see examples_)

Utilities:

* Subscription - check if user is subscribed to the channels of interest.
* Middleware - some helpful middleware functions.
* Helper functions for logging, etc.

Breaking Changes in V4
======================

Controls now operate on Interfaces defined in `interface.go` rather than functions.
There's a new convenience structure TVC which can be used to wrap the functions when updating to v4.

See examples_ for usage.

Installation
============

For Telebot_ v3::
  
  go get github.com/rusq/tbcomctl/v4

For Telebot_ v2::
  
  go get github.com/rusq/tbcomctl
  // or
  go get github.com/rusq/tbcomctl/v2

v2 is not actively developed, but you're more than welcome to submit your PRs.

Usage
=====
For usage - see examples_.



.. _Telebot: https://github.com/tucnak/telebot
.. _examples: examples
