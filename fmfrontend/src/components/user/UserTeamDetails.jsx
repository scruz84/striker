import React from "react";
import {Card, CardContent, Grid} from "@mui/material";
import Typography from "@mui/material/Typography";
import Box from "@mui/material/Box";
import {CircularDeterminateWithLabel, LinearDeterminateWithLabel} from "../common/_common";
import PropTypes from 'prop-types';

export function UserTeamDetails({userTeam}) {
    return (
        <Box >
            <Card sx={{ minWidth: 275 }}>
                <CardContent>
                    <Typography variant="h6" gutterBottom>
                        {userTeam.team.name}
                    </Typography>
                </CardContent>
            </Card>
            {userTeam.players.map((player) => (
                <PlayerDetails key={player.id} player={player}/>
            ))}
        </Box>
    );
}

function PlayerDetails({player}) {
    return (
        <Card sx={{ minWidth: 275,  maxHeight: 300 }}>
            <CardContent >
                <Grid container spacing={2}>
                    <Grid item xs={4} md={2} >
                        <img height="100" src="avatars/player.svg" loading="lazy" alt={player.firstName}/>
                    </Grid>

                    <Grid item xs={6} md={12}>
                        <Typography variant="body2">
                            {player.firstName} {player.lastName}
                        </Typography>
                    </Grid>
                </Grid>
                <Grid item xs={6} md={4}>
                    <SkillValue name="Goalkeeper" value={player.goalKeeper} />
                    <SkillValue name="Defense" value={player.defense} />
                    <SkillValue name="Lateral" value={player.lateral} />
                    <SkillValue name="Pass" value={player.pass} />
                    <SkillValue name="Midfield" value={player.midfield} />
                    <SkillValue name="Dribbling" value={player.dribbling} />
                    <SkillValue name="Shot" value={player.shot} />
                </Grid>
            </CardContent>
        </Card>
    );
}

SkillValue.propTypes = {
    name: PropTypes.string.isRequired,
    value: PropTypes.number.isRequired,
};

function SkillValue(props) {
    return (
        <Grid container spacing={1} sx={{ml:2}}>
            <Grid item xs={2} md={2}>
                <Typography variant="body2">
                    {props.name}
                </Typography>
            </Grid>
            <Grid item xs={9} md={6}>
                <LinearDeterminateWithLabel value={props.value} />
            </Grid>
        </Grid>
    );
}
