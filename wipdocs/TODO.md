# todo list

## remote control

- Remote control has a bug when controlling another web session where the current track info does not update even though skip/previous/play/pause etc all work. 

## misc

- [ ] css needs a lot of cleanup, can we move all of it to scss
- [ ] media session is buggy and doesn't seem to always update, especially when create a new queue not sure whats up
- [ ] mobile friendly page is not working at all
- [ ] Update the backend to create a hash of the file in the db so it can skip scanning if it already exists

- [x] Searching with playlists does not work
- [ ] Show recommended playlist suggestions by searching for matching tracks not in current playlist, by trying to match artist, album, and genres found within tags if possible.
- [ ] Update web/music/cmd/musicd/templates.templ and web/music/cmd/musicd/main.go to support a configurable path prefix so instead of `/static` or `/playlists` we can make this have an optional prefix like `/myprefix/static` or `/myprefix/playlists`. This is very useful in some reverse proxy setups
- [x] Update web/music/cmd/musicd/templates.templ so that audio player bar so that controls are always centered and allow the left album/artist view to "fill" the space
- [x] Update web/music/cmd/musicd/templates.templ so that audio player bar so that controls are aware of the screen width so that it can stack the track info on top of the controls on top of the volume control to make display better on smaller screens
- [ ] Update the navbar that when you click on the logo it takes you back to the all tracks view
- [ ] Update web/music/cmd/musicd/templates.templ so that we have a mobile friendly navbar, that has an entry do display All Tracks, an entry to Rescan Library including a confirmation dialog and a submenu to display Playlists. The Playlists submenu should have Create at the top and then show all available Playlists for the user to navigate to.
- [ ] Update web/music/cmd/musicd/templates.templ so that our navbar has submenu for Settings that includes the action to rescan the library. The Playlists submenu should have Create at the top and then show all available Playlists for the user to navigate to however currently it is not showing all available playlists and only shows the create action.
- [ ] Update web/music/cmd/musicd/templates.templ and web/music/cmd/musicd/main.go to have pages for viewing Artists, and under each Artist show All available Albums. Under Each artist we should show all tracks below the albums and have pages for the album which shows all tracks under that album
- [x] Update the playlist view to be a dropdown or something similar
- [x] when you select a playlist to play, it queues all tracks in the library instead of just those in the playlist with no search provided
- [ ] Duration for tracks missing
- [ ] Update web/music/cmd/musicd/templates.templ to use just a single top level search bar instead of having multiple search bars for tracks/playlists/albums/artists etc...
- [ ] Pagination doesn't work reliably when navigating around


- [ ] Update backend to store audio metadata in postgres. During a scan we should get a list of tracks, then add new ones, finally we should remove any tracks that are missing in the database. For cover art we should store all cover art in a seperate table and have each song just reference that postgres id. That way for the frontend we can save some data by reducing duplicate cover images being pushed to the frontend.

- [ ] Update web/music/cmd/musicd/main.go so that all the routes are under /api/ and serve up json data instead of htmx templates and then Migrate web/music/cmd/musicd/templates.templ into web/music/frontend that uses vue/bulma for the pages. I have already migrated the css into web/music/frontend/assets/css/site.css and the user state is now under web/music/frontend/stores/appState.ts. 

- [ ] Update artists and albums pages:
- Git rid of rounded album cover displays
- Play All, Add All To Queue needs fixed
- Track list when playing needs to always fetch all data without pagination to produce queue!
- Randomize needs to happen outside the queue, and same goes for the previous track. We want a FIFO to keep track of the last 200 songs or something.


Update playground/web/music/cmd/musicd so that it uses postgres to store the music library into. I already created playground/web/music/init.sql that contains our database schema. When we scan the library we should be aware of what tracks already are present so that we can remove missing and add new. I split the cover art into a different table because when we request all the songs from the backend we need to keep the cover art seperate or else the object grows too large.

Update web/music/cmd/musicd/main.go so that artists and albums are also inserted into postgres instead of making up ids at runtime. I already created playground/web/music/init.sql that has the artist/album tables

## 20250907

- Playlists no longer work
- [ ] Update web/music/cmd/musicd/main.go and web/music/frontend/app/ so that playlists are properly parsed and loaded into our database. All playlists should be reference by id and not name. All playlists should have a route for `/api/playlist/{id}` and `/api/playlist/{id}/tracks` that supports pagination. When we add/remove a track from a playlist we should update the database AND the m3u file on disk.
- [ ] Update /playground/web/music/frontend/app/layouts/default.vue so that it includes a button in the bottom control bar to display the visualizer component at playground/web/music/frontend/app/components/WaveformBackground.vue as a fullscreen overlay.

- We want a way to view the WaveformBackground visual as well by including `<WaveformBackground :audioEl="audioRef!" :fixed="true" :showDiagnostics="false" />`
