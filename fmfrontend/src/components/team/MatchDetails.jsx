import {useParams} from "react-router-dom";
import {useDispatch, useSelector} from "react-redux";
import React, {useEffect} from "react";
import {matchDetailsReset} from "../../redux/actions";
import {GetMatchDetails} from "../user/_teamFunctions";
import Box from "@mui/material/Box";
import {Divider, Grid, Paper, Skeleton, Tab, Tabs} from "@mui/material";
import Moment from "react-moment";
import Typography from "@mui/material/Typography";
import PropTypes from 'prop-types';
import SportsSoccerIcon from '@mui/icons-material/SportsSoccer';

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

function TabPanel(props) {
    const { children, value, index, ...other } = props;
    return (
        <div
            role="tabpanel"
            hidden={value !== index}
            id={`simple-tabpanel-${index}`}
            aria-labelledby={`simple-tab-${index}`}
            {...other}>
            {value === index && (
                <Box sx={{ p: 3 }}>
                    <Typography>{children}</Typography>
                </Box>
            )}
        </div>
    );
}

TabPanel.propTypes = {
    children: PropTypes.node,
    index: PropTypes.number.isRequired,
    value: PropTypes.number.isRequired,
};

function a11yProps(index) {
    return {
        id: `simple-tab-${index}`,
        'aria-controls': `simple-tabpanel-${index}`,
    };
}

function PlayedMatchDetails(m) {
    const [value, setValue] = React.useState(0);
    const handleChange = (event, newValue) => {
        setValue(newValue);
    };

    console.log(m.matchDetails)
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
                    <Box sx={{ borderBottom: 1, borderColor: 'divider' }}>
                        <Tabs value={value} onChange={handleChange} variant="scrollable" scrollButtons="auto">
                            <Tab label="Eventos" {...a11yProps(0)} wrapped/>
                            <Tab label="Resumen" {...a11yProps(1)} />
                            <Tab label="Calificaciones" {...a11yProps(2)} />
                            <Tab label="Jugadores" {...a11yProps(3)} />
                        </Tabs>
                    </Box>
                    <TabPanel value={value} index={0}>
                        <Events events={m.matchDetails.events}/>
                    </TabPanel>
                    <TabPanel value={value} index={1}>
                        Resumen
                    </TabPanel>
                    <TabPanel value={value} index={2}>
                        Calificaciones
                    </TabPanel>
                    <TabPanel value={value} index={3}>
                        Jugadores...!
                    </TabPanel>
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

function Events(events) {
    return (
        <div>
            {events.events.filter(event => event.eventType === 0).map((event) => (
                <Grid container spacing={0} columns={10}>
                    <Grid xs={1}>
                        <SportsSoccerIcon/>
                    </Grid>
                    <Grid xs={1}>
                        <Divider orientation="vertical" />
                    </Grid>
                    <Grid xs={6} sx={{ml: 1}}>
                        {event.teamOwnerPlayers}
                    </Grid>
                    <Grid xs={1}>
                        {event.minute}'
                    </Grid>
                </Grid>
            ))}
        </div>
    )
}
