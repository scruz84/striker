import Breadcrumbs from '@mui/material/Breadcrumbs';
import Stack from '@mui/material/Stack';
import NavigateNextIcon from '@mui/icons-material/NavigateNext';
import {useSelector} from "react-redux";
import React from "react";
import {Link} from "react-router-dom";

export default function Subheader() {
    const userReducer = useSelector((state) => state.userReducer);

    if (userReducer.userTeam) {
        const breadcrumbs = [
            <Link to='/' style={{ textDecoration: 'none', color: 'inherit' }}>{userReducer.userTeam.team.name}</Link>,
            <Link to={`/division/${userReducer.userTeam.team.divisionId}`}
                  style={{textDecoration: 'none', color: 'inherit'}}>
                {userReducer.userTeam.team.divisionName}
            </Link>
        ];

        return (
            <Stack spacing={2}>
                <Breadcrumbs
                    separator={<NavigateNextIcon fontSize="small"/>}
                    aria-label="breadcrumb">
                    {breadcrumbs}
                </Breadcrumbs>
            </Stack>
        );
    }
}
