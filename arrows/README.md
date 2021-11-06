# arrows
A minimal game of remove the arrows

Click on the arrows to remove them from the screen (if they have a free path in the direction they are pointing to).

## Usage:

    arrows [-width=#] [-height=n] [-audio=true/false] [-term=true/false] [-shuffle=random/left/right]

 - width: number of columns
 - height: number of rows
 - audio: enable/disable audio
 - term: "terminal" UI vs. graphics UI
 - shuffle: shuffle direction

By default you'll see the graphical UI (based on gio) but you can use the terminal version by passing the "-term" option.

You can build a browser based version using the command `gogio -target js .` (it requires `gogio` from `gioui.org/cmd/gogio` to be installed) or you can use the provided Makefile:

    make js

The Makefile provides other useful target:

    # install desktop game
    make install
    
    # build desktop game
    make build
    
    # build ios game (not working yet)
    make ios
    
    # build android game (not tested)
    make android
    
    # remove all generated files
    make clean

## Mouse commands:
 - move mouse: move cursor
 - click: move/remove arrow

## Keyboard commands:

 - up, down, left, right arrow: move cursor
 - space: move/remove arrow

 - U/u: Undo last move
 - R/r: reset game
 - S/s: reshuffle game
 - H/h: help/hint
 - P/p: autoplay

