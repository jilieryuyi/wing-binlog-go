<?php
/**
 * Created by PhpStorm.
 * User: yuyi
 * Date: 17/11/10
 * Time: 22:55
 */
$json = "{
\"database\":\"new_yonglibao_c\",
\"event\":{
    \"data\":{
        \"new_data\":{
            \"id\":1,
            \"user_id\":523,
            \"invest_id\":11,
            \"payout_money\":100.00000,
            \"payback_at\":1416887067,
            \"days\":0,
            \"affirm_money\":100.00000,
            \"created\":1416887067,
            \"pay_type\":1
            },
        \"old_data\":{
            \"id\":1,
            \"user_id\":523,
            \"invest_id\":112,
            \"payout_money\":100.00000,
            \"payback_at\":1416887067,
            \"days\":0,
            \"affirm_money\":100.00000,
            \"created\":1416887067,
            \"pay_type\":1
        }},\"event_type\":\"update\",\"time\":1510325639},\"event_index\":253131300,\"table\":\"bw_active_payout\"}";
var_dump(json_decode($json, true));