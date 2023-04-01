import Box from "@mui/material/Box";
import AppBar from "@mui/material/AppBar";
import Toolbar from "@mui/material/Toolbar";
import IconButton from "@mui/material/IconButton";
import MenuIcon from "@mui/icons-material/Menu";
import Typography from "@mui/material/Typography";
import React from "react";
import {Link, useNavigate} from "react-router-dom";
import {Avatar, Menu, MenuItem, Tooltip} from "@mui/material";
import {useDispatch, useSelector} from "react-redux";
import {DoLogout} from "./Login";

const settings = ['Profile', 'Account', 'Dashboard'];

function Header() {
    return (
        <Box sx={{ flexGrow: 1 }}>
            <AppBar position="static">
                <Toolbar>
                    <IconButton
                        size="large"
                        edge="start"
                        color="inherit"
                        aria-label="menu"
                        sx={{ mr: 2 }}
                    >
                        <MenuIcon />
                    </IconButton>
                    <Typography variant="h6" component="div" sx={{ flexGrow: 1 }}>
                        <Link to='/' style={{ textDecoration: 'none', color: 'inherit' }}>Striker</Link>
                    </Typography>
                    <AvatarOrLogin/>
                </Toolbar>
            </AppBar>
        </Box>
    );
}


function AvatarOrLogin() {
    const nameReducer = useSelector((state) => state.userReducer);
    const dispatch = useDispatch();
    const navigate = useNavigate();
    const loggedUser = nameReducer.loggedUser;
    const [anchorElNav, setAnchorElNav] = React.useState(null);
    const [anchorElUser, setAnchorElUser] = React.useState(null);

    const handleOpenNavMenu = (event) => {
        setAnchorElNav(event.currentTarget);
    };
    const handleOpenUserMenu = (event) => {
        setAnchorElUser(event.currentTarget);
    };

    const handleCloseNavMenu = () => {
        setAnchorElNav(null);
    };

    const handleCloseUserMenu = () => {
        setAnchorElUser(null);
    };

    const handleLogout = () => {
        DoLogout(dispatch, navigate, '/');
        setAnchorElUser(null);
    }

    return (
        loggedUser == null?
            <Link color="inherit" to='/login' underline="hover">Login</Link>
            : <Box sx={{ flexGrow: 0 }}>
                <Tooltip title="Open settings">
                    <IconButton onClick={handleOpenUserMenu} sx={{ p: 0 }}>
                        <Avatar src={loggedUser.pictureUrl} referrerPolicy="no-referrer" />
                    </IconButton>
                </Tooltip>
                <Menu
                    sx={{ mt: '45px' }}
                    id="menu-appbar"
                    anchorEl={anchorElUser}
                    anchorOrigin={{
                        vertical: 'top',
                        horizontal: 'right',
                    }}
                    keepMounted
                    transformOrigin={{
                        vertical: 'top',
                        horizontal: 'right',
                    }}
                    open={Boolean(anchorElUser)}
                    onClose={handleCloseUserMenu}
                >
                    {settings.map((setting) => (
                        <MenuItem key={setting} onClick={handleCloseUserMenu}>
                            <Typography textAlign="center">{setting}</Typography>
                        </MenuItem>
                    ))}
                    <MenuItem key='Logout' onClick={handleLogout}>
                        <Typography textAlign="center">Logout</Typography>
                    </MenuItem>

                </Menu>
            </Box>
    );
}

export default Header;