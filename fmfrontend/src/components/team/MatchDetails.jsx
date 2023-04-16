import {useParams} from "react-router-dom";
import {useDispatch, useSelector} from "react-redux";
import React, {useEffect} from "react";
import {matchDetailsReset} from "../../redux/actions";
import {GetMatchDetails} from "../user/_teamFunctions";
import Box from "@mui/material/Box";
import {Grid, Paper, Skeleton, Tab, Tabs} from "@mui/material";
import Moment from "react-moment";
import Typography from "@mui/material/Typography";

export function MatchDetails() {
    const routeParams = useParams();
    const dispatch = useDispatch()
    const matchReducer = useSelector((state) => state.userReducer);
    const matchDetails = matchReducer.matchDetails;

    useEffect(() => {
        // Anything in here is fired on component mount.
        if (!matchDetails) {
            GetMatchDetails(dispatch, routeParams.id)
        }
        return () => {
            // Anything in here is fired on component unmount.
            dispatch(matchDetailsReset());
        }
    }, [])

    if (matchReducer.matchDetails) {
        return (
            (matchReducer.matchDetails.played?
                <PlayedMatchDetails matchDetails = {matchReducer.matchDetails}/>
                : <NotPlayedMatchDetails matchDetails = {matchReducer.matchDetails}/>)
        );
    } else {
        return (
            <Skeleton variant="rectangular" height={200}/>
        );
    }
}

function PlayedMatchDetails(m) {
    const [value, setValue] = React.useState('Eventos');
    const handleChange = (event, newValue) => {
        setValue(newValue);
    };

    return (
        <div>
            <Paper>
                <Grid container spacing={0}>
                    <Grid item xs={2} />
                    <Grid item xs={8}>
                        <Moment format="DD/MM/YYYY HH:mm">
                            {m.matchDetails.matchDate}
                        </Moment>
                    </Grid>
                    <Grid item xs={2} />

                    <Grid item xs={5} >
                        <Typography fontSize="small" xs={1}>
                            {m.matchDetails.homeName}
                        </Typography>
                    </Grid>

                    <Grid item xs={2}>
                        <Typography fontSize="small" xs={1}>
                            {m.matchDetails.homeGoals} - {m.matchDetails.awayGoals}
                        </Typography>
                    </Grid>
                    <Grid item xs={5} >
                        <Typography fontSize="small" xs={1}>
                            {m.matchDetails.awayName}
                        </Typography>
                    </Grid>
                </Grid>
            </Paper>
            <Paper>
                <Box sx={{ width: '100%' }}>
                    <Tabs value={value} onChange={handleChange} variant="scrollable" scrollButtons="auto">
                        <Tab value="Eventos" label="Eventos"  wrapped/>
                        <Tab value="Resumen" label="Resumen" />
                        <Tab value="Calificaciones" label="Calificaciones" />
                        <Tab value="Jugadores" label="Jugadores" />
                    </Tabs>
                </Box>
            </Paper>
        </div>
    );
}

function NotPlayedMatchDetails(matchDetails) {
    return (
        <div>Not Played!</div>
    );
}