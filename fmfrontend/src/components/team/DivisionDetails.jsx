import {Link, useParams} from "react-router-dom";
import {useDispatch, useSelector} from "react-redux";
import React, {useEffect} from "react";
import {divisionDetailsReset} from "../../redux/actions";
import {GetDivisionDetails} from "../user/_divisionFunctions";
import Moment from 'react-moment';
import CalendarTodayTwoToneIcon from '@mui/icons-material/CalendarTodayTwoTone';
import {
    Grid,
    List,
    ListItem,
    Paper,
    Table,
    TableBody,
    TableCell,
    TableContainer,
    TableHead,
    TableRow
} from "@mui/material";
import Box from "@mui/material/Box";
import Typography from "@mui/material/Typography";
import Stack from "@mui/material/Stack";

export function DivisionDetails() {
    const routeParams = useParams();

    const dispatch = useDispatch()
    const divReducer = useSelector((state) => state.userReducer);
    const divisionDetails = divReducer.divisionDetails;

    useEffect(() => {
        // Anything in here is fired on component mount.
        if (!divisionDetails) {
            GetDivisionDetails(dispatch, routeParams.id)
        }
        return () => {
            // Anything in here is fired on component unmount.
            dispatch(divisionDetailsReset());
        }
    }, [])

    if (divReducer.divisionDetails) {
        const classificationTable = GetClassificationTable(divisionDetails)
        return (
            <div>
                <h3>Jornada {divisionDetails.classification.positions[0].matches}</h3>
                <Box /*sx={{ width: 800}}*/>
                    {classificationTable}
                </Box>
            </div>
        );
    } else {
        return ( <div>loading...</div> );
    }
}

function GetClassificationTable(divisionDetails) {

    return (
        <div>
            <TableContainer component={Paper}>
                <Table size="small" aria-label="Classification">
                    <TableHead>
                        <TableRow>
                            <TableCell>#</TableCell>
                            <TableCell align="left">Equipo</TableCell>
                            <TableCell align="left">P</TableCell>
                            <TableCell align="left">Avg</TableCell>
                            <TableCell align="left">G</TableCell>
                            <TableCell align="left">E</TableCell>
                            <TableCell align="left">P</TableCell>
                            <TableCell align="left">+</TableCell>
                            <TableCell align="left">-</TableCell>
                        </TableRow>
                    </TableHead>
                    <TableBody>
                        {divisionDetails.classification.positions.map((row) => (
                            <TableRow
                                key={row.position}
                                sx={{ '&:last-child td, &:last-child th': { border: 0 } }}>
                                <TableCell component="th" scope="row">{row.position}</TableCell>
                                <TableCell align="left">
                                    <Link to={`/team/${row.teamId}`} style={{ color: 'inherit' }}>
                                        {row.teamName}</Link>
                                </TableCell>
                                <TableCell align="center">{row.points}</TableCell>
                                <TableCell align="left">{row.goalsScored-row.goalsReceived}</TableCell>
                                <TableCell align="left">{row.wins}</TableCell>
                                <TableCell align="left">{row.draws}</TableCell>
                                <TableCell align="left">{row.losses}</TableCell>
                                <TableCell align="left">{row.goalsScored}</TableCell>
                                <TableCell align="left">{row.goalsReceived}</TableCell>
                            </TableRow>
                        ))}
                    </TableBody>
                </Table>
            </TableContainer>
            <Paper>
                <h4>Última semana</h4>
                <PrintWeekMatches weekMatches={divisionDetails.previousWeekMatches}/>
            </Paper>
            <Paper>
                <h4>Próxima semana</h4>
                <PrintWeekMatches weekMatches={divisionDetails.nextWeekMatches}/>
            </Paper>
        </div>
    );
}

function PrintWeekMatches(weekMatches) {
    return (
        <List>
            <Grid container spacing={0}>
                {weekMatches.weekMatches.map((row, idx) => (
                    <Link to={`/match/${row.matchId}`} style={{color: 'inherit'}}>
                        <ListItem key={idx} alignItems="flex-start">
                            <Grid item xs={9} >
                                <Typography fontSize="small" xs={1}>
                                    {row.homeName} - {row.awayName}
                                </Typography>
                                <Stack direction="row">
                                    <CalendarTodayTwoToneIcon fontSize="small"/>
                                    <Typography fontSize="small" xs={0}>
                                        <Moment format="DD/MM/YYYY HH:mm">
                                             {row.matchDate}
                                        </Moment>
                                    </Typography>
                                </Stack>
                            </Grid>
                            <Grid item xs={2}>
                                <Typography fontSize="small" xs={0}>
                                    {row.homeGoals} - {row.awayGoals}</Typography>
                            </Grid>
                        </ListItem>
                    </Link>
                ))}
            </Grid>
        </List>
    );
}