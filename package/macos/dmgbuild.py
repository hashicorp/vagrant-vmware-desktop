# Copyright IBM Corp. 2021, 2025
# SPDX-License-Identifier: MPL-2.0

from __future__ import unicode_literals

import biplist
import os.path

# Default contents

format = defines.get('format', 'UDZO')
size = defines.get('size', '102400k')
files_dir = defines.get('srcfolder', '.')
files = [
    "{}/VagrantVMwareUtility.pkg".format(files_dir),
    "{}/uninstall.tool".format(files_dir)
]

# Set the background

background = defines.get('backgroundimg', 'builtin-arrow')

# Hide things we don't want to see

show_status_bar = False
show_tab_view = False
show_toolbar = False
show_pathbar = False
show_sidebar = False

# Set size and view style

window_rect = ((100, 100), (605, 540))
default_view = 'icon-view'

# Arrange contents

arrange_by = None
icon_size = 72
grid_offset = (0, 0)
grid_spacing = 100
scroll_position = (0, 0)
label_pos = 'bottom'
text_size = 14
show_icon_preview = False
icon_locations = {
    'VagrantVMwareUtility.pkg': (420, 60),
    'uninstall.tool': (420, 220)
}
