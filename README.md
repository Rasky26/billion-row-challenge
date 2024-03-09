# Billion Row Challenge
### Two Week Challenge with the TurnKey Correction Team

> ## Ideas
> * Open file with multiple readers
> * Utilize all cores
> * Have each core build its own map?
>   * And after each map is built, run a function that combines them into a singular output map
> * Use integers instead of floats
> * Channels with buffers?
> * Probably don't use mutexes as that could slow down the writing? Or is that a required steps based on how I build my stuff?